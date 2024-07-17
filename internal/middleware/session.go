package middleware

import (
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/ghaggin/sso/internal/config"
	"github.com/ghaggin/sso/internal/model"
	"go.uber.org/fx"
)

const (
	sessionKey       = "session_key"
	redirectStateKey = "redirect_state_key"
)

var (
	errSessionNotFound = errors.New("session not found")
)

type SessionManager struct {
	impl *scs.SessionManager
}

type SessionManagerParams struct {
	fx.In

	Mode config.Mode
}

func NewSessionManager(p SessionManagerParams) (*SessionManager, error) {
	gob.Register(&model.Session{})
	gob.Register(&http.Request{})
	gob.Register(&http.NoBody)

	sm := &SessionManager{}
	sm.impl = scs.New()
	sm.impl.Cookie = scs.SessionCookie{
		Name:     fmt.Sprintf("%s_%s", p.Mode, getUniqueSuffix()),
		Domain:   "",
		HttpOnly: true,
		Path:     "/",
		Persist:  true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	}

	return sm, nil
}

func (s *SessionManager) Wrap(next http.Handler) http.Handler {
	return s.impl.LoadAndSave(next)
}

func (s *SessionManager) Get(ctx context.Context) (*model.Session, error) {
	session, ok := s.impl.Get(ctx, sessionKey).(*model.Session)
	if !ok {
		return nil, errSessionNotFound
	}

	return session, nil
}

func (s *SessionManager) SetAuthenticated(ctx context.Context, name string) error {
	session, ok := s.impl.Get(ctx, sessionKey).(*model.Session)
	if !ok {
		session = &model.Session{}
	}

	session.UID = name
	session.AuthValid = true
	session.AuthExpiration = time.Now().Add(time.Minute)

	s.impl.Put(ctx, sessionKey, session)
	return nil
}

func (s *SessionManager) StoreLoginRedirectState(ctx context.Context, r *http.Request) {
	s.impl.Put(ctx, redirectStateKey, r)
}

func (s *SessionManager) LoadLoginRedirectState(ctx context.Context) (*http.Request, bool) {
	r, ok := s.impl.Get(ctx, redirectStateKey).(*http.Request)
	return r, ok
}

func getUniqueSuffix() string {
	charSet := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	n := len(charSet)
	var s string
	for len(s) < 7 {
		i := rand.Intn(n)
		s += string(charSet[i])
	}
	return s
}
