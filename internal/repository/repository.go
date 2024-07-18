package repository

import (
	"context"
	"errors"

	"github.com/ghaggin/sso/internal/model"
)

var (
	ErrNotFound = errors.New("not found")
)

type Repository interface {
	GetUserByName(ctx context.Context, name string) (*model.User, error)
	AddUser(ctx context.Context, user *model.User) error
	GetUsers(ctx context.Context) ([]model.User, error)
}
