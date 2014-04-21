package httpsession

import (
	"net/http"
	"sync"
	"time"
)

type Id string

type Store interface {
	Get(id Id, key string) interface{}
	Set(id Id, key string, value interface{})
	Del(id Id, key string) bool
	Clear(id Id) bool
	Add(id Id)
	Exist(id Id) bool
	Run() error
}

type Session struct {
	id      Id
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

const (
	DefaultExpireTime = 30 * time.Minute
)

type Manager struct {
	store                  Store
	MaxAge                 int
	Path                   string
	generator              IdGenerator
	transfer               Transfer
	beforeReleaseListeners map[BeforeReleaseListener]bool
	afterCreatedListeners  map[AfterCreatedListener]bool
	lock                   sync.Mutex
}

func Default() *Manager {
	store := NewMemoryStore(DefaultExpireTime)
	key := string(GenRandKey(16))
	return NewManager(store,
		NewSha1Generator(key),
		NewCookieTransfer("SESSIONID", DefaultExpireTime))
}

func NewManager(store Store, gen IdGenerator, transfer Transfer) *Manager {
	return &Manager{
		store:     store,
		generator: gen,
		transfer:  transfer,
	}
}

func (manager *Manager) Session(req *http.Request, rw http.ResponseWriter) *Session {
	manager.lock.Lock()
	defer manager.lock.Unlock()

	id, err := manager.transfer.Get(req)
	if err != nil {
		// TODO:
		println("error:", err.Error())
		return nil
	}

	if !manager.generator.IsValid(id) /*|| !manager.store.Exist(id)*/ {
		id = manager.generator.Gen(req)
		manager.transfer.Set(req, rw, id)
		manager.store.Add(id)
	}

	session := &Session{id: id, manager: manager}
	// is exist?
	manager.afterCreated(session)
	return session
}

func (manager *Manager) Invalidate(rw http.ResponseWriter, session *Session) {
	manager.beforeReleased(session)
	manager.store.Clear(session.id)
	manager.transfer.Clear(rw)
}

func (manager *Manager) afterCreated(session *Session) {
	for listener, _ := range manager.afterCreatedListeners {
		listener.OnAfterCreated(session)
	}
}

func (manager *Manager) beforeReleased(session *Session) {
	for listener, _ := range manager.beforeReleaseListeners {
		listener.OnBeforeRelease(session)
	}
}

func (manager *Manager) Run() error {
	return manager.store.Run()
}
