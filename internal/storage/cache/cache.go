package cache

import (
    "time"
    "errors"

    "github.com/bluele/gcache"
)

// gcache item wrapper
type AppLRUCache struct {
    c *gcache.Cache
}

// base cache interface
type MemCache interface {
    Get(key string) (interface{}, error)
    Set(key string, val interface{}) (bool, error)
    Setex(key string, val interface{}, exp time.Duration) (bool, error)
    // callback when item evicked from cache
    OnEvict(func(key interface{}) error) error
}

// create new LRU cache
func NewLRUCache(size int) (*AppLRUCache, error) {
    //...
    if size <= 0 {
        return &AppLRUCache{}, errors.New("Cache size <= 0")
    }
    c := gcache.New(size).LRU().Build()
    return &AppLRUCache{&c}, nil
}
