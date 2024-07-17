package idp

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/crewjam/saml/samlidp"
	"github.com/ghaggin/sso/internal/config"
	"github.com/go-chi/chi/v5"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type IdentityProvider struct {
	log    *zap.Logger
	server *http.Server
}

type Params struct {
	fx.In

	Log    *zap.Logger
	Config *config.Config
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

	// Setup Router
	root := chi.NewRouter()
	root.Get("/metadata", func(w http.ResponseWriter, r *http.Request) {
		idpServer.IDP.ServeMetadata(w, r)
	})
	root.HandleFunc("/sso", func(w http.ResponseWriter, r *http.Request) {
		idpServer.IDP.ServeSSO(w, r)
	})

	root.Get("/service", idpServer.HandleGetService)
	root.Put("/service", idpServer.HandlePutService)
	root.Post("/service", idpServer.HandlePutService)

	return &IdentityProvider{
		log: p.Log,
		server: &http.Server{
			Addr:    fmt.Sprintf("localhost:%d", p.Config.IdentityProvider.Port),
			Handler: root,
		},
	}, nil
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
