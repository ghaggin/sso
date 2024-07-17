package idp

import (
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Controller struct {
	log *zap.Logger
}

type ControllerParams struct {
	fx.In

	Logger *zap.Logger
}

func NewController(p ControllerParams) (*Controller, error) {
	return &Controller{
		log: p.Logger,
	}, nil
}

func (c *Controller) ValidateLogin(u string, p string) (bool, error) {
	if u == "glen.haggin" && p == "a" {
		return true, nil
	}
	return false, nil
}
