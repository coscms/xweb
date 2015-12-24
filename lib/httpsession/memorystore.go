package httpsession

import (
	"sync"
	"time"
)

var _ Store = NewMemoryStore(30)

type sessionNode struct {
	lock   sync.RWMutex
	Kvs    map[string]interface{}
	Last   time.Time
	MaxAge time.Duration
}

func (node *sessionNode) Get(key string) interface{} {
	node.lock.RLock()
	v := node.Kvs[key]
	node.lock.RUnlock()
	node.lock.Lock()
	node.Last = time.Now()
	node.lock.Unlock()
	return v
}

func (node *sessionNode) Set(key string, v interface{}) {
	node.lock.Lock()
	node.Kvs[key] = v
	node.Last = time.Now()
	node.lock.Unlock()
}

func (node *sessionNode) Del(key string) {
	node.lock.Lock()
	delete(node.Kvs, key)
	node.Last = time.Now()
	node.lock.Unlock()
}

type MemoryStore struct {
	lock       sync.RWMutex
	nodes      map[Id]*sessionNode
	GcInterval time.Duration
	maxAge     time.Duration
}

func NewMemoryStore(maxAge time.Duration) *MemoryStore {
	return &MemoryStore{nodes: make(map[Id]*sessionNode),
		maxAge: maxAge, GcInterval: 10 * time.Second}
}

func (store *MemoryStore) SetMaxAge(maxAge time.Duration) {
	store.lock.Lock()
	store.maxAge = maxAge
	store.lock.Unlock()
}

func (store *MemoryStore) Get(id Id, key string) interface{} {
	store.lock.RLock()
	node, ok := store.nodes[id]
	store.lock.RUnlock()
	if !ok {
		return nil
	}

	if store.maxAge > 0 && time.Now().Sub(node.Last) > node.MaxAge {
		// lazy DELETE expire
		store.lock.Lock()
		delete(store.nodes, id)
		store.lock.Unlock()
		return nil
	}

	return node.Get(key)
}

func (store *MemoryStore) Set(id Id, key string, value interface{}) {
	store.lock.RLock()
	node, ok := store.nodes[id]
	store.lock.RUnlock()
	if !ok {
		store.lock.Lock()
		node = &sessionNode{Kvs: make(map[string]interface{}),
			Last:   time.Now(),
			MaxAge: store.maxAge,
		}
		node.Kvs[key] = value
		store.nodes[id] = node
		store.lock.Unlock()
	}

	node.Set(key, value)
}

func (store *MemoryStore) Add(id Id) {
	node := &sessionNode{Kvs: make(map[string]interface{}), Last: time.Now()}
	store.lock.Lock()
	store.nodes[id] = node
	store.lock.Unlock()
}

func (store *MemoryStore) Del(id Id, key string) bool {
	store.lock.RLock()
	node, ok := store.nodes[id]
	store.lock.RUnlock()
	if ok {
		node.Del(key)
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
		store.GC()
		store.Run()
	})
	return nil
}

//随机检查过期时间
func (store *MemoryStore) GC() {
	store.lock.Lock()
	defer store.lock.Unlock()
	if store.maxAge == 0 {
		return
	}
	var i, j int
	for k, v := range store.nodes {
		if j > 20 || i > 5 {
			break
		}
		if time.Now().Sub(v.Last) > v.MaxAge {
			delete(store.nodes, k)
			i = i + 1
		}
		j = j + 1
	}

}
