package repository

import (
	"context"
	"encoding/json"
	"errors"
	"os"

	"github.com/ghaggin/sso/internal/config"
	"github.com/ghaggin/sso/internal/model"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var (
	errTableFileIsDir = errors.New("table file is dir")
)

type Data struct {
	Users []model.User `json:"users"`
}

type jsonRepo struct {
	path string
	log  *zap.Logger

	data *Data
}

type jsonParams struct {
	fx.In

	LC     fx.Lifecycle
	Config *config.Config
	Log    *zap.Logger
}

func NewJSON(p jsonParams) (Repository, error) {
	r := &jsonRepo{
		path: p.Config.JSONRepo.Path,
		log:  p.Log,
		data: &Data{},
	}

	err := r.readfile()
	if err != nil {
		// only log, data will be empty and will overwrite when
		// the service is stopped
		r.log.Warn("failed reading json repo data file", zap.Error(err))
	}

	p.LC.Append(fx.Hook{
		OnStop: r.stop,
	})

	return r, nil
}

func (r *jsonRepo) stop(_ context.Context) error {
	return r.writefile()
}

func (r *jsonRepo) readfile() error {
	finfo, err := os.Stat(r.path)
	if err != nil {
		return err
	}

	if finfo.IsDir() {
		return errTableFileIsDir
	}

	f, err := os.Open(r.path)
	if err != nil {
		return err
	}

	return json.NewDecoder(f).Decode(&r.data)
}

func (r *jsonRepo) writefile() error {
	f, err := os.Create(r.path)
	if err != nil {
		return err
	}

	b, err := json.MarshalIndent(r.data, "", "  ")
	if err != nil {
		return err
	}

	_, err = f.Write(b)
	return err
}

func (r *jsonRepo) GetUserByName(_ context.Context, name string) (*model.User, error) {
	for _, u := range r.data.Users {
		if u.Name == name {
			return &u, nil
		}
	}

	return nil, ErrNotFound
}

func (r *jsonRepo) AddUser(_ context.Context, user *model.User) error {
	user.ID = 0
	l := len(r.data.Users)
	if l > 0 {
		user.ID = r.data.Users[l-1].ID + 1
	}

	r.data.Users = append(r.data.Users, *user)
	return nil
}

func (r *jsonRepo) GetUsers(_ context.Context) ([]model.User, error) {
	return r.data.Users, nil
}
