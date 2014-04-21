package httpsession

import (
	"sync"
	"time"
)

var _ Store = NewMemoryStore(30)

type sessionNode struct {
	lock sync.RWMutex
	kvs  map[string]interface{}
	last time.Time
}

type MemoryStore struct {
	lock       sync.RWMutex
	nodes      map[Id]*sessionNode
	expire     time.Duration
	GcInterval time.Duration
}

func NewMemoryStore(expire time.Duration) *MemoryStore {
	return &MemoryStore{nodes: make(map[Id]*sessionNode),
		expire: expire, GcInterval: 10 * time.Second}
}

func (store *MemoryStore) Get(id Id, key string) interface{} {
	store.lock.RLock()
	node, ok := store.nodes[id]
	store.lock.RUnlock()
	if !ok {
		return nil
	}

	if time.Now().Sub(node.last) > store.expire {
		// lazy DELETE expire
		store.lock.Lock()
		delete(store.nodes, id)
		store.lock.Unlock()
		return nil
	}

	node.lock.Lock()
	node.last = time.Now()
	node.lock.Unlock()

	node.lock.RLock()
	v, ok := node.kvs[key]
	node.lock.RUnlock()

	if !ok {
		return nil
	}
	return v
}

func (store *MemoryStore) Set(id Id, key string, value interface{}) {
	store.lock.Lock()
	node, ok := store.nodes[id]
	if !ok {
		node = &sessionNode{kvs: make(map[string]interface{}), last: time.Now()}
		node.kvs[key] = value
		store.nodes[id] = node
		store.lock.Unlock()
	} else {
		store.lock.Unlock()
		node.lock.Lock()
		node.last = time.Now()
		node.kvs[key] = value
		node.lock.Unlock()
	}
}

func (store *MemoryStore) Add(id Id) {
	node := &sessionNode{kvs: make(map[string]interface{}), last: time.Now()}
	store.lock.Lock()
	store.nodes[id] = node
	store.lock.Unlock()
}

func (store *MemoryStore) Del(id Id, key string) bool {
	store.lock.RLock()
	node, ok := store.nodes[id]
	store.lock.RUnlock()
	if ok {
		node.lock.Lock()
		delete(node.kvs, key)
		node.lock.Unlock()
	}
	return true
}

func (store *MemoryStore) Exist(id Id) bool {
	store.lock.RLock()
	defer store.lock.RUnlock()
	_, ok := store.nodes[id]
	return ok
}

func (store *MemoryStore) Clear(id Id) bool {
	store.lock.Lock()
	defer store.lock.Unlock()
	delete(store.nodes, id)
	return true
}

func (store *MemoryStore) Run() error {
	time.AfterFunc(store.GcInterval, func() {
		store.Run()
		store.GC()
	})
	return nil
}

//
func (store *MemoryStore) GC() {
	store.lock.Lock()
	defer store.lock.Unlock()
	var i = 0
	for k, v := range store.nodes {
		if i > 20 {
			break
		}
		if time.Now().Sub(v.last) > store.expire {
			store.lock.Lock()
			delete(store.nodes, k)
			store.lock.Unlock()
			i = i + 1
		}
	}
}
