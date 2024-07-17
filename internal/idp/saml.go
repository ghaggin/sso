package idp

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/crewjam/saml"
	"github.com/crewjam/saml/logger"
	"github.com/crewjam/saml/samlidp"
	xrv "github.com/mattermost/xml-roundtrip-validator"
	dsig "github.com/russellhaering/goxmldsig"
)

type SamlIdentityProvider struct {
	IDP             *saml.IdentityProvider
	serviceProvider *saml.EntityDescriptor
}

func newSamlIdentityProvider(opts samlidp.Options) *SamlIdentityProvider {
	metadataURL := opts.URL
	metadataURL.Path += "/metadata"
	ssoURL := opts.URL
	ssoURL.Path += "/sso"
	logoutURL := opts.URL
	logoutURL.Path += "/logout"

	logr := opts.Logger
	if logr == nil {
		logr = logger.DefaultLogger
	}

	validDuration := time.Hour * 24 * 2

	s := &SamlIdentityProvider{
		IDP: &saml.IdentityProvider{
			Key:             opts.Key,
			Signer:          opts.Signer,
			Certificate:     opts.Certificate,
			Intermediates:   nil,
			SignatureMethod: dsig.RSASHA1SignatureMethod,

			ValidDuration: &validDuration,

			Logger: logr,

			MetadataURL: metadataURL,
			SSOURL:      ssoURL,
			LogoutURL:   logoutURL,

			ServiceProviderProvider: nil, // set below
			SessionProvider:         nil, // set below
			AssertionMaker:          saml.DefaultAssertionMaker{},
		},
	}

	s.IDP.ServiceProviderProvider = s
	s.IDP.SessionProvider = s

	return s
}

func (s *SamlIdentityProvider) GetSession(w http.ResponseWriter, r *http.Request, req *saml.IdpAuthnRequest) *saml.Session {
	return &saml.Session{
		ID:                    "1",
		CreateTime:            time.Now().Add(-time.Hour),
		ExpireTime:            time.Now().Add(time.Hour),
		Index:                 "2",
		NameID:                "3",
		NameIDFormat:          "4",
		SubjectID:             "5",
		Groups:                []string{"a", "b"},
		UserName:              "6",
		UserEmail:             "7",
		UserCommonName:        "8",
		UserSurname:           "9",
		UserGivenName:         "10",
		UserScopedAffiliation: "11",
		CustomAttributes:      []saml.Attribute{},
	}
}

func (s *SamlIdentityProvider) GetServiceProvider(_ *http.Request, serviceProviderID string) (*saml.EntityDescriptor, error) {
	return s.serviceProvider, nil
}

func (s *SamlIdentityProvider) HandlePutService(w http.ResponseWriter, r *http.Request) {
	metadata, err := getSPMetadata(r.Body)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
	s.serviceProvider = metadata
}

func (s *SamlIdentityProvider) HandleGetService(w http.ResponseWriter, r *http.Request) {
	err := xml.NewEncoder(w).Encode(s.serviceProvider)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func getSPMetadata(r io.Reader) (spMetadata *saml.EntityDescriptor, err error) {
	var data []byte
	if data, err = io.ReadAll(r); err != nil {
		return nil, err
	}

	spMetadata = &saml.EntityDescriptor{}
	if err := xrv.Validate(bytes.NewBuffer(data)); err != nil {
		return nil, err
	}

	if err := xml.Unmarshal(data, &spMetadata); err != nil {
		if err.Error() == "expected element type <EntityDescriptor> but have <EntitiesDescriptor>" {
			entities := &saml.EntitiesDescriptor{}
			if err := xml.Unmarshal(data, &entities); err != nil {
				return nil, err
			}

			for _, e := range entities.EntityDescriptors {
				if len(e.SPSSODescriptors) > 0 {
					return &e, nil
				}
			}

			// there were no SPSSODescriptors in the response
			return nil, errors.New("metadata contained no service provider metadata")
		}

		return nil, err
	}

	return spMetadata, nil
}
