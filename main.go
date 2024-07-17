package main

import (
	"flag"

	"github.com/ghaggin/sso/internal/config"
	"github.com/ghaggin/sso/internal/idp"
	"github.com/ghaggin/sso/internal/middleware"
	"github.com/ghaggin/sso/internal/sp"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func main() {
	var mode = flag.String("mode", "", "either sp or idp")
	flag.Parse()

	newMode := func() config.Mode {
		return config.Mode(*mode)
	}

	deps := fx.Options(
		fx.Provide(
			zap.NewDevelopment,
			config.New,
			middleware.NewSessionManager,
			newMode,
		),
	)

	var app *fx.App
	if *mode == "sp" {
		app = fx.New(
			deps,
			fx.Provide(sp.New),
			fx.Invoke(sp.RegisterHooks),
		)
	} else if *mode == "idp" {
		app = fx.New(
			deps,
			idp.Module,
			fx.Invoke(idp.RegisterHooks),
		)
	} else {
		panic("unrecognized mode")
	}

	app.Run()
}
