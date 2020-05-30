package config

import "sync"

var once sync.Once
var instance *sync.Map

func init() {
	once.Do(func() {
		instance = new(sync.Map)
	})
}

func Load(key string) (interface{}, bool) {
	return instance.Load(key)
}

func Store(key string, value interface{}) {
	instance.Store(key, value)
}

func LoadOrStore(key string, value interface{}) (interface{}, bool) {
	return instance.LoadOrStore(key, value)
}

func Range(f func(key, value interface{}) bool) {
	instance.Range(f)
}

func Delete(key interface{}) {
	instance.Delete(key)
}
