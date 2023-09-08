package cache

import (
    "fmt"
    "time"
    "errors"

    "github.com/bluele/gcache"

    "nats_app/internal/config"
)

var (
    EmptyCacheKey = errors.New("Key can`t be empty string")
)

// base cache interface
type MemCache interface {
    Get(key string) (interface{}, error)
    Set(key string, val interface{}) (bool, error)
    Setex(key string, val interface{}, exp time.Duration) (bool, error)
    // callback when item evicked from cache
    OnEvict(func(interface{}, interface{})) MemCache
    // callback when item added to cache
    OnAdd(func(string, interface{})) MemCache
    OnLoad(func(string) interface{}) MemCache
    Build() MemCache
}

// gcache item wrapper
type AppLRUCache struct {
    c *gcache.Cache
    ExpT time.Duration
    Size int
    On_load func(string) interface{}
    on_evict func(string, []byte)
    on_add func(string, []byte)
}

func (ac *AppLRUCache) Get(key string) (interface{}, error) {
    var val interface{}
    var err error
    if val, err = (*ac.c).Get(key); err != nil {
        return nil, fmt.Errorf("%w", err)
    }
    return val, nil
}

func (ac *AppLRUCache) Set(key string, val interface{}) (bool, error) {
    mark := "AppLRUCache.Set"
    if key == "" {
        return false, EmptyCacheKey
    }
    err := (*ac.c).Set(key, val)
    if err != nil {
        return false, fmt.Errorf("%s error %w", mark, err)
    }
    return true, nil
}

func (ac *AppLRUCache) Setex(key string, val interface{}, exp time.Duration) (bool, error) {
    mark := "AppLRUCache.Setex"
    if key == "" {
        return false, EmptyCacheKey
    }
    err := (*ac.c).SetWithExpire(key, val, exp)
    if err != nil {
        return false, fmt.Errorf("%s error %w", mark, err)
    }
    return true, nil
}

func (ac *AppLRUCache) OnEvict(evict func(string, []byte)) *AppLRUCache {
    (*ac).on_evict = evict
    return ac
}

func (ac *AppLRUCache) OnAdd(add func(string, []byte)) *AppLRUCache {
    (*ac).on_add = add
    return ac
}

// set handler for automatically data fetching from DB.
func (ac *AppLRUCache) OnLoad(loader func(string) interface{}) *AppLRUCache {
    (*ac).On_load = loader
    return ac
}

func (ac *AppLRUCache) Build() *AppLRUCache {
    c := gcache.New((*ac).Size).
        LRU().
        EvictedFunc(func(key, value interface{}) {
            (*ac).on_evict(key.(string), value.([]byte))
        }).
        AddedFunc(func(key, value interface{}) {
            (*ac).on_add(key.(string), value.([]byte))
        }).
        Build()
    (*ac).c = &c
    return ac
}

// create new LRU cache
func NewLRUCache(conf *config.CacheConfig) *AppLRUCache {
    return &AppLRUCache{Size: (*conf).Size, ExpT: (*conf).Exp_time}
}
