package httpsession

import (
	"encoding/gob"
	"log"
	"time"

	"github.com/coscms/xweb/lib/str"
	"github.com/bradfitz/gomemcache/memcache"
)

var RegNodeToGob bool

type MemcacheStore struct {
	c      *memcache.Client
	maxAge time.Duration
	Debug  bool
}

func NewMemcacheStore(maxAge time.Duration, conn []string) *MemcacheStore {
	if !RegNodeToGob {
		gob.Register(&sessionNode{})
	}
	return &MemcacheStore{c: memcache.New(conn...), maxAge: maxAge}
}

func (store *MemcacheStore) SetMaxAge(maxAge time.Duration) {
	store.maxAge = maxAge
}

func (store *MemcacheStore) Get(id Id, key string) interface{} {
	v := store.get(id)
	if v == nil {
		return nil
	}
	return v.Get(key)
}

func (store *MemcacheStore) Set(id Id, key string, value interface{}) {
	v := store.get(id)
	if v == nil {
		v = &sessionNode{Kvs: make(map[string]interface{}),
			Last:   time.Now(),
			MaxAge: store.maxAge,
		}
	}
	v.Set(key, value)
	store.set(id, v)
}

func (store *MemcacheStore) Add(id Id) {
	store.set(id, &sessionNode{Kvs: make(map[string]interface{}),
		Last:   time.Now(),
		MaxAge: store.maxAge,
	})
}

func (store *MemcacheStore) get(id Id) *sessionNode {
	key := string(id)
	val, err := store.c.Get(key)
	if err != nil || val == nil {
		if err != nil && store.Debug {
			log.Println("[Memcache]GetErr: ", err, "Key:", key)
		}
		return nil
	}

	var v interface{}
	err = str.Decode(val.Value, &v)
	if err != nil {
		if store.Debug {
			log.Println("[Memcache]DecodeErr: ", err, "Key:", key)
		}
		return nil
	}
	return v.(*sessionNode)
}
func (store *MemcacheStore) set(id Id, v *sessionNode) bool {
	key := string(id)
	val, err := str.Encode(v)
	if err != nil {
		if store.Debug {
			log.Println("[Memcache]EncodeErr: ", err, "Key:", key)
		}
		return false
	}
	item := &memcache.Item{Key: key, Value: val, Expiration: int32(store.maxAge.Seconds())}
	//println(item.Expiration)
	err = store.c.Set(item)
	if err != nil {
		if store.Debug {
			log.Println("[Memcache]PutErr: ", err, "Key:", key)
		}
		return false
	}
	if store.Debug {
		log.Println("[Memcache]Put: ", v, "Key", key)
	}
	return true
}
func (store *MemcacheStore) Del(id Id, key string) bool {
	v := store.get(id)
	if v == nil {
		return true
	}
	v.Del(key)
	return store.set(id, v)
}

func (store *MemcacheStore) Exist(id Id) bool {
	key := string(id)
	mk := str.Md5(key)
	val, err := store.c.Get(mk)
	if err != nil || val == nil {
		return false
	}
	return true
}

func (store *MemcacheStore) Clear(id Id) bool {
	key := string(id)
	err := store.c.Delete(str.Md5(key))
	if err != nil {
		if store.Debug {
			log.Println("[Memcache]DelErr: ", err, "Key:", key)
		}
		return false
	}
	if store.Debug {
		log.Println("[Memcache]Del: ", key)
	}
	return true
}

func (store *MemcacheStore) Run() error {
	return nil
}
