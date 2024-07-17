package idp

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/crewjam/saml/samlidp"
	"github.com/ghaggin/sso/internal/config"
	"github.com/ghaggin/sso/internal/middleware"
	"github.com/ghaggin/sso/internal/model"
	"github.com/ghaggin/sso/internal/template"
	"github.com/go-chi/chi/v5"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type IdentityProvider struct {
	log    *zap.Logger
	server *http.Server
	sm     *middleware.SessionManager
	ctrl   *Controller
}

type Params struct {
	fx.In

	Log            *zap.Logger
	Config         *config.Config
	SessionManager *middleware.SessionManager
	Controller     *Controller
}

func New(p Params) (*IdentityProvider, error) {
	// Build baseurl from port
	baseUrl, err := url.Parse(fmt.Sprintf("http://localhost:%d", p.Config.IdentityProvider.Port))
	if err != nil {
		return nil, err
	}

	// Get certificates
	// TODO: move to internal/config module, provide as dependency
	key, cert, err := config.GetKeyPair("idp")
	if err != nil {
		return nil, err
	}

	// Setup IDP server
	idpServer := newSamlIdentityProvider(samlidp.Options{
		URL:         *baseUrl,
		Key:         key,
		Certificate: cert,
		Store:       &samlidp.MemoryStore{},
	})

	idp := &IdentityProvider{
		log: p.Log,
		sm:  p.SessionManager,
		server: &http.Server{
			Addr: fmt.Sprintf("localhost:%d", p.Config.IdentityProvider.Port),
		},
	}

	// Setup Router
	root := chi.NewRouter()
	root.Use(idp.sm.Wrap)

	root.Group(func(r chi.Router) {
		r.Use(idp.requireAuth)
		r.HandleFunc("/", idp.home)
		r.HandleFunc("/sso", idpServer.IDP.ServeSSO)
	})

	root.Get("/login", idp.getLogin)
	root.Post("/login", idp.putLogin)
	root.Get("/redirect", idp.redirect)

	root.Get("/metadata", func(w http.ResponseWriter, r *http.Request) {
		idpServer.IDP.ServeMetadata(w, r)
	})

	root.Get("/service", idpServer.HandleGetService)
	root.Put("/service", idpServer.HandlePutService)
	root.Post("/service", idpServer.HandlePutService)

	root.Handle("/static/*", http.StripPrefix("/static", http.FileServer(http.Dir("web/static/"))))

	idp.server.Handler = root
	return idp, nil
}

// RegisterHooks should be invoked by fx
func RegisterHooks(lc fx.Lifecycle, i *IdentityProvider) {
	lc.Append(fx.Hook{
		OnStart: i.Start,
		OnStop:  i.server.Shutdown,
	})
}

func (i *IdentityProvider) Start(_ context.Context) error {
	go func() {
		err := i.server.ListenAndServe()
		if err != nil {
			i.log.Error("error starting server", zap.Error(err))
		}
	}()
	return nil
}

func (i *IdentityProvider) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := i.sm.Get(r.Context())
		if err != nil || !session.AuthValid || time.Now().After(session.AuthExpiration) {
			i.sm.StoreLoginRedirectState(r.Context(), r)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (i *IdentityProvider) home(w http.ResponseWriter, r *http.Request) {
	err := template.Render(w, r, "idp/home.html", &template.Data{
		PageTitle: "home",
	})
	if err != nil {
		i.log.Error("error rendering idp_home.html", zap.Error(err))
		i.renderError(w, r, err)
	}
}

func (i *IdentityProvider) getLogin(w http.ResponseWriter, r *http.Request) {
	err := template.Render(w, r, "idp/login.html", &model.IDPLoginData{
		BaseData: model.BaseData{
			PageTitle: "Login",
		},
	})
	if err != nil {
		i.log.Error("error rendering idp_login.html", zap.Error(err))
		i.renderError(w, r, err)
	}
}

func (i *IdentityProvider) putLogin(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	username := r.Form.Get("username")
	password := r.Form.Get("password")

	i.log.Info("login attempt", zap.String("username", username), zap.String("password", password))
	valid, err := i.ctrl.ValidateLogin(username, password)

	renderErr := false
	renderErrMsg := ""
	if err != nil {
		renderErr = true
		renderErrMsg = err.Error()
	} else if !valid {
		renderErr = true
		renderErrMsg = "username and/or password failed validation"
	}

	if renderErr {
		i.log.Error("failed login", zap.String("error", renderErrMsg))

		err := template.Render(w, r, "idp/login.html", &model.IDPLoginData{
			BaseData: model.BaseData{
				PageTitle: "Login",
			},
			Error:        true,
			ErrorMessage: renderErrMsg,
		})
		if err != nil {
			i.renderError(w, r, err)
		}
		return
	}

	err = i.sm.SetAuthenticated(r.Context(), username)
	if err != nil {
		i.renderError(w, r, err)
		return
	}

	http.Redirect(w, r, "/redirect", http.StatusSeeOther)
}

func (i *IdentityProvider) redirect(w http.ResponseWriter, r *http.Request) {
	if rr, found := i.sm.LoadLoginRedirectState(r.Context()); found {
		for _, cookie := range r.Cookies() {
			rr.AddCookie(cookie)
		}
		i.server.Handler.ServeHTTP(w, rr)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

func (i *IdentityProvider) renderError(w http.ResponseWriter, _ *http.Request, err error) {
	w.Write([]byte(err.Error()))
}
