package sp

import (
	"context"
	"encoding/gob"
	"fmt"
	"net/http"
	"time"

	scs "github.com/alexedwards/scs/v2"
	"github.com/crewjam/saml/samlsp"
	"github.com/ghaggin/sso/internal/config"
	"github.com/ghaggin/sso/internal/template"
	"github.com/go-chi/chi/v5"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ServiceProvider struct {
	log    *zap.Logger
	server *http.Server
}

type Params struct {
	fx.In

	Log    *zap.Logger
	Config *config.Config
}

func New(p Params) (*ServiceProvider, error) {
	samlSP, err := NewSAML(fmt.Sprintf(":%d", p.Config.ServiceProvider.Port), "http://localhost:8124/metadata")
	if err != nil {
		panic(err)
	}

	newSessionManager()

	root := chi.NewRouter()
	root.Use(sessionManager.LoadAndSave)

	// Auth
	root.Group(func(r chi.Router) {
		r.Use(requireAuth)
		r.Get("/", home)
		r.Get("/attr", getAttrVals)
	})

	// No Auth
	root.Group(func(r chi.Router) {
		r.HandleFunc("/login", login)
		r.HandleFunc("/saml/login", func(w http.ResponseWriter, r *http.Request) {
			samlSP.HandleStartAuthFlow(w, r)
		})

		r.Get("/saml/metadata", samlSP.ServeMetadata)
		r.Post("/saml/acs", samlSP.ServeACS)

		r.Handle("/static/*", http.StripPrefix("/static", http.FileServer(http.Dir("web/static/"))))
	})

	return &ServiceProvider{
		log: p.Log,
		server: &http.Server{
			Addr:    fmt.Sprintf("localhost:%d", p.Config.ServiceProvider.Port),
			Handler: root,
		},
	}, nil
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

var sessionManager *scs.SessionManager

func newSessionManager() {
	gob.Register(&User{})

	sessionManager = scs.New()
	sessionManager.Lifetime = time.Minute * 3
}

func home(w http.ResponseWriter, r *http.Request) {
	user, ok := sessionManager.Get(r.Context(), "user").(*User)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}

	template.Render(w, r, "home.html", &template.Data{
		PageTitle: "home",
		UID:       user.UID,
	})
}

func login(w http.ResponseWriter, r *http.Request) {
	template.Render(w, r, "login.html", &template.Data{
		PageTitle: "login",
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

type User struct {
	UID string
}

// Presence of a user in the context indicates auth
func requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := sessionManager.Get(r.Context(), "user").(*User)
		if !ok {
			http.Redirect(w, r, "/saml/login", http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	})
}
