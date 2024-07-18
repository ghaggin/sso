package idp

import (
	"context"

	"github.com/ghaggin/sso/internal/model"
	"github.com/ghaggin/sso/internal/repository"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Controller struct {
	repo repository.Repository
	log  *zap.Logger
}

type ControllerParams struct {
	fx.In

	Logger *zap.Logger
	Repo   repository.Repository
}

func NewController(p ControllerParams) (*Controller, error) {
	return &Controller{
		log:  p.Logger,
		repo: p.Repo,
	}, nil
}

func (c *Controller) ValidateLogin(ctx context.Context, username string, password string) (bool, error) {
	u, err := c.repo.GetUserByName(ctx, username)
	if err != nil {
		return false, err
	}

	if u.Password == password {
		return true, nil
	}

	return false, nil
}

func (c *Controller) CreateUser(ctx context.Context, user *model.User) error {
	return c.repo.AddUser(ctx, user)
}

func (c *Controller) GetUsers(ctx context.Context) ([]model.User, error) {
	return c.repo.GetUsers(ctx)
}
