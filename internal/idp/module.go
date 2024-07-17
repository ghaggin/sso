package idp

import "go.uber.org/fx"

var Module = fx.Options(
	fx.Provide(
		New,
		NewController,
	),
)
