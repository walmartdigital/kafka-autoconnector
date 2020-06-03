package cache

import "sync"

// Cache ...
type Cache interface {
	Load(key string) (interface{}, bool)
	Store(key string, value interface{})
	LoadOrStore(key string, value interface{}) (interface{}, bool)
	Range(f func(key, value interface{}) bool)
	Delete(key interface{})
}

// InMemoryCache ...
type InMemoryCache struct {
	instance *sync.Map
}

// NewInMemoryCache ...
func NewInMemoryCache() *InMemoryCache {
	obj := InMemoryCache{
		instance: new(sync.Map),
	}
	return &obj
}

// Load ...
func (c InMemoryCache) Load(key string) (interface{}, bool) {
	return c.instance.Load(key)
}

// Store ...
func (c InMemoryCache) Store(key string, value interface{}) {
	c.instance.Store(key, value)
}

// LoadOrStore ...
func (c InMemoryCache) LoadOrStore(key string, value interface{}) (interface{}, bool) {
	return c.instance.LoadOrStore(key, value)
}

// Range ...
func (c InMemoryCache) Range(f func(key, value interface{}) bool) {
	c.instance.Range(f)
}

// Delete ...
func (c InMemoryCache) Delete(key interface{}) {
	c.instance.Delete(key)
}
