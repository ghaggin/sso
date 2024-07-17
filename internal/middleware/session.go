package middleware

import (
	"context"
	"encoding/gob"
	"errors"
	"net/http"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/ghaggin/sso/internal/model"
)

const (
	sessionKey = "session_key"
)

var (
	errSessionNotFound = errors.New("session not found")
)

type SessionManager struct {
	impl *scs.SessionManager
}

func NewSessionManager() (*SessionManager, error) {
	gob.Register(&model.Session{})

	sm := &SessionManager{}
	sm.impl = scs.New()

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
