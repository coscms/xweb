package httpsession

import (
	"net/http"
	"time"
)

type Session struct {
	id      Id
	maxAge  time.Duration
	manager *Manager
}

func (session *Session) Id() Id {
	return session.id
}

func (session *Session) Get(key string) interface{} {
	return session.manager.store.Get(session.id, key)
}

func (session *Session) Set(key string, value interface{}) {
	session.manager.store.Set(session.id, key, value)
}

func (session *Session) Del(key string) bool {
	return session.manager.store.Del(session.id, key)
}

func (session *Session) Invalidate(rw http.ResponseWriter) {
	session.manager.Invalidate(rw, session)
}

func (session *Session) IsValid() bool {
	return session.manager.generator.IsValid(session.id)
}

func (session *Session) SetMaxAge(maxAge time.Duration) {
	session.maxAge = maxAge
}

func NewSession(id Id, maxAge time.Duration, manager *Manager) *Session {
	return &Session{id: id, maxAge: manager.maxAge, manager: manager}
}
