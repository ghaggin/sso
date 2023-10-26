package internal

import (
	"bytes"
	"context"
	"encoding/xml"
	"net/http"
	"net/url"

	"github.com/crewjam/saml"
	"github.com/crewjam/saml/samlsp"
	dsig "github.com/russellhaering/goxmldsig"
)

type SamlServiceProvider interface {
	ServeMetadata(w http.ResponseWriter, _ *http.Request)
	ServeACS(w http.ResponseWriter, r *http.Request)
	HandleStartAuthFlow(w http.ResponseWriter, r *http.Request)
}

type samlServiceProvider struct {
	ServiceProvider *saml.ServiceProvider
	Binding         string
	ResponseBinding string
	OnError         func(w http.ResponseWriter, r *http.Request, err error)
	RequestTracker  samlsp.RequestTracker
}

func newSamlSP(port, idpURL string) (SamlServiceProvider, error) {
	idpMetadataURL, err := url.Parse(idpURL)
	if err != nil {
		return nil, err
	}
	idpMetadata, err := samlsp.FetchMetadata(context.Background(), http.DefaultClient, *idpMetadataURL)
	if err != nil {
		return nil, err
	}

	rootURL, err := url.Parse("http://localhost" + port)
	if err != nil {
		return nil, err
	}

	key, cert, err := getKeyPair("sp")
	if err != nil {
		return nil, err
	}

	opts := samlsp.Options{
		EntityID:          "test_sp",
		URL:               *rootURL,
		Key:               key,
		Certificate:       cert,
		IDPMetadata:       idpMetadata,
		AllowIDPInitiated: true,
	}

	var forceAuthn *bool
	if opts.ForceAuthn {
		forceAuthn = &opts.ForceAuthn
	}

	if opts.DefaultRedirectURI == "" {
		opts.DefaultRedirectURI = "/"
	}

	if len(opts.LogoutBindings) == 0 {
		opts.LogoutBindings = []string{saml.HTTPPostBinding}
	}

	samlSP := &samlServiceProvider{
		Binding:         "",
		ResponseBinding: saml.HTTPPostBinding,
		OnError:         samlsp.DefaultOnError,
		ServiceProvider: &saml.ServiceProvider{
			EntityID:              opts.EntityID,
			Key:                   opts.Key,
			Certificate:           opts.Certificate,
			HTTPClient:            opts.HTTPClient,
			Intermediates:         opts.Intermediates,
			MetadataURL:           *opts.URL.ResolveReference(&url.URL{Path: "saml/metadata"}),
			AcsURL:                *opts.URL.ResolveReference(&url.URL{Path: "saml/acs"}),
			SloURL:                *opts.URL.ResolveReference(&url.URL{Path: "saml/slo"}),
			IDPMetadata:           opts.IDPMetadata,
			ForceAuthn:            forceAuthn,
			RequestedAuthnContext: opts.RequestedAuthnContext,
			SignatureMethod:       dsig.RSASHA1SignatureMethod,
			AllowIDPInitiated:     opts.AllowIDPInitiated,
			DefaultRedirectURI:    opts.DefaultRedirectURI,
			LogoutBindings:        opts.LogoutBindings,
		}}

	samlSP.RequestTracker = samlsp.DefaultRequestTracker(opts, samlSP.ServiceProvider)
	if opts.UseArtifactResponse {
		samlSP.ResponseBinding = saml.HTTPArtifactBinding
	}

	return samlSP, nil
	// return samlsp.New(opts)
}

func (s *samlServiceProvider) ServeMetadata(w http.ResponseWriter, _ *http.Request) {
	buf, err := xml.MarshalIndent(s.ServiceProvider.Metadata(), "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/samlmetadata+xml")
	if _, err := w.Write(buf); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *samlServiceProvider) ServeACS(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		s.OnError(w, r, err)
		return
	}

	possibleRequestIDs := []string{}
	if s.ServiceProvider.AllowIDPInitiated {
		possibleRequestIDs = append(possibleRequestIDs, "")
	}

	trackedRequests := s.RequestTracker.GetTrackedRequests(r)
	for _, tr := range trackedRequests {
		possibleRequestIDs = append(possibleRequestIDs, tr.SAMLRequestID)
	}

	assertion, err := s.ServiceProvider.ParseResponse(r, possibleRequestIDs)
	if err != nil {
		s.OnError(w, r, err)
		return
	}

	// TODO: handle assertion
	for _, as := range assertion.AttributeStatements {
		for _, a := range as.Attributes {
			if a.FriendlyName == "uid" && len(a.Values) == 1 {
				sessionManager.Put(r.Context(), "user", &User{UID: a.Values[0].Value})
			}
		}
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

func (s *samlServiceProvider) HandleStartAuthFlow(w http.ResponseWriter, r *http.Request) {
	var binding, bindingLocation string
	if s.Binding != "" {
		binding = s.Binding
		bindingLocation = s.ServiceProvider.GetSSOBindingLocation(binding)
	} else {
		binding = saml.HTTPRedirectBinding
		bindingLocation = s.ServiceProvider.GetSSOBindingLocation(binding)
		if bindingLocation == "" {
			binding = saml.HTTPPostBinding
			bindingLocation = s.ServiceProvider.GetSSOBindingLocation(binding)
		}
	}

	authReq, err := s.ServiceProvider.MakeAuthenticationRequest(bindingLocation, binding, s.ResponseBinding)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// relayState is limited to 80 bytes but also must be integrity protected.
	// this means that we cannot use a JWT because it is way to long. Instead
	// we set a signed cookie that encodes the original URL which we'll check
	// against the SAML response when we get it.
	relayState, err := s.RequestTracker.TrackRequest(w, r, authReq.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if binding == saml.HTTPRedirectBinding {
		redirectURL, err := authReq.Redirect(relayState, s.ServiceProvider)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Add("Location", redirectURL.String())
		w.WriteHeader(http.StatusFound)
		return
	}
	if binding == saml.HTTPPostBinding {
		w.Header().Add("Content-Security-Policy", ""+
			"default-src; "+
			"script-src 'sha256-AjPdJSbZmeWHnEc5ykvJFay8FTWeTeRbs9dutfZ0HqE='; "+
			"reflected-xss block; referrer no-referrer;")
		w.Header().Add("Content-type", "text/html")
		var buf bytes.Buffer
		buf.WriteString(`<!DOCTYPE html><html><body>`)
		buf.Write(authReq.Post(relayState))
		buf.WriteString(`</body></html>`)
		if _, err := w.Write(buf.Bytes()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	}
	panic("not reached")
}
