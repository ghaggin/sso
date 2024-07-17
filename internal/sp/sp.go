package sp

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/crewjam/saml/samlsp"
	"github.com/ghaggin/sso/internal/config"
	"github.com/ghaggin/sso/internal/middleware"
	"github.com/ghaggin/sso/internal/template"
	"github.com/go-chi/chi/v5"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ServiceProvider struct {
	log    *zap.Logger
	server *http.Server
	sm     *middleware.SessionManager
}

type Params struct {
	fx.In

	Log            *zap.Logger
	Config         *config.Config
	SessionManager *middleware.SessionManager
}

func New(p Params) (*ServiceProvider, error) {
	samlSP, err := NewSAML(fmt.Sprintf(":%d", p.Config.ServiceProvider.Port), "http://localhost:8124/metadata", p.SessionManager)
	if err != nil {
		panic(err)
	}

	sp := &ServiceProvider{
		log: p.Log,
		server: &http.Server{
			Addr: fmt.Sprintf("localhost:%d", p.Config.ServiceProvider.Port),
		},
		sm: p.SessionManager,
	}

	root := chi.NewRouter()
	root.Use(p.SessionManager.Wrap)

	// Auth
	root.Group(func(r chi.Router) {
		r.Use(sp.requireAuth)
		r.Get("/", sp.home)
		r.Get("/attr", getAttrVals)
	})

	// No Auth
	root.Group(func(r chi.Router) {
		r.HandleFunc("/login", sp.login)
		r.HandleFunc("/saml/login", func(w http.ResponseWriter, r *http.Request) {
			samlSP.HandleStartAuthFlow(w, r)
		})

		r.Get("/saml/metadata", samlSP.ServeMetadata)
		r.Post("/saml/acs", samlSP.ServeACS)

		r.Handle("/static/*", http.StripPrefix("/static", http.FileServer(http.Dir("web/static/"))))
	})

	sp.server.Handler = root
	return sp, nil
}

func RegisterHooks(lc fx.Lifecycle, s *ServiceProvider) {
	lc.Append(fx.Hook{
		OnStart: s.Start,
		OnStop:  s.server.Shutdown,
	})
}

func (s *ServiceProvider) Start(_ context.Context) error {
	go func() {
		err := s.server.ListenAndServe()
		if err != nil {
			s.log.Error("error shutting down server", zap.Error(err))
		}
	}()
	return nil
}

func (s *ServiceProvider) home(w http.ResponseWriter, r *http.Request) {
	session, err := s.sm.Get(r.Context())
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}

	template.Render(w, r, "home.html", &template.Data{
		PageTitle: "home",
		UID:       session.UID,
	})
}

func (s *ServiceProvider) login(w http.ResponseWriter, r *http.Request) {
	template.Render(w, r, "login.html", &template.Data{
		PageTitle: "login",
	})
}

// Presence of a user in the context indicates auth
func (s *ServiceProvider) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := s.sm.Get(r.Context())
		if err != nil || !session.AuthValid || time.Now().After(session.AuthExpiration) {
			http.Redirect(w, r, "/saml/login", http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func getAttrVals(w http.ResponseWriter, r *http.Request) {
	s := samlsp.SessionFromContext(r.Context())
	if s == nil {
		fmt.Fprint(w, "s is null")
		return
	}

	sa, ok := s.(samlsp.SessionWithAttributes)
	if !ok {
		fmt.Fprint(w, "couldn't cast sa")
		return
	}
	attrs := sa.GetAttributes()

	for attr, attrVal := range attrs {
		fmtAttrVal := ""
		for i, v := range attrVal {
			if i == 0 {
				fmtAttrVal = v
				continue
			}
			fmtAttrVal = fmtAttrVal + ", " + v
		}
		fmt.Fprintf(w, "%v: %v\n", attr, fmtAttrVal)
	}
}
