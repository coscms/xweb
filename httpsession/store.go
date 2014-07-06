package httpsession

import "time"

type Id string

type Store interface {
	Get(id Id, key string) interface{}
	Set(id Id, key string, value interface{})
	Del(id Id, key string) bool
	Clear(id Id) bool
	Add(id Id)
	Exist(id Id) bool
	SetMaxAge(maxAge time.Duration)
	Run() error
}
