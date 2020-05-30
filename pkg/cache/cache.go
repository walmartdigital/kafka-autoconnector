package cache

import "sync"

type Cache interface {
	Load(key string) (interface{}, bool)
	Store(key string, value interface{})
	LoadOrStore(key string, value interface{}) (interface{}, bool)
	Range(f func(key, value interface{}) bool)
	Delete(key interface{})
}

type InMemoryCache struct {
	instance *sync.Map
}

func NewInMemoryCache() *InMemoryCache {
	obj := InMemoryCache{
		instance: new(sync.Map),
	}
	return &obj
}

func (c InMemoryCache) Load(key string) (interface{}, bool) {
	return c.instance.Load(key)
}

func (c InMemoryCache) Store(key string, value interface{}) {
	c.instance.Store(key, value)
}

func (c InMemoryCache) LoadOrStore(key string, value interface{}) (interface{}, bool) {
	return c.instance.LoadOrStore(key, value)
}

func (c InMemoryCache) Range(f func(key, value interface{}) bool) {
	c.instance.Range(f)
}

func (c InMemoryCache) Delete(key interface{}) {
	c.instance.Delete(key)
}
