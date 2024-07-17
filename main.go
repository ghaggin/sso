package main

import (
	"flag"

	"github.com/ghaggin/sso/internal/config"
	"github.com/ghaggin/sso/internal/idp"
	"github.com/ghaggin/sso/internal/sp"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func main() {
	var mode = flag.String("mode", "", "either sp or idp")
	flag.Parse()

	deps := fx.Options(
		fx.Provide(
			zap.NewDevelopment,
			config.New,
			// idp.New,
			// sp.New,
		),
	)

	var app *fx.App
	if *mode == "sp" {
		app = fx.New(
			deps,
			fx.Provide(sp.New),
			fx.Invoke(sp.RegisterHooks),
		)
	} else {
		app = fx.New(
			deps,
			fx.Provide(idp.New),
			fx.Invoke(idp.RegisterHooks),
		)
	}

	app.Run()
}
