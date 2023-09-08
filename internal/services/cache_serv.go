package services

import (
    "fmt"
    "errors"
    "context"
    "log/slog"
    "sync"
    "time"

    "nats_app/internal/storage/cache"
)

const (
    AddOne string = "add_one"
    AddMany string = "add_many"
    Evicted uint8 = 1
    Added uint8 = 0
)

type ConcurrentLog interface {
    Dump(context.Context, chan<- interface{})
    LogEvicted(val interface{}) error
    LogAdded(val interface{}) error
}

type LogMessage interface {
    OpCode() uint8
    Payload() string
}

type CacheLogMessage struct {
    // operation: evict or add
    op uint8
    // key to fing record in DB
    key string
}

func (clm CacheLogMessage) OpCode() uint8 {
    return clm.op
}

func (clm CacheLogMessage) Payload() string {
    return clm.key
}

type CacheLog struct {
    lock sync.RWMutex
    records []CacheLogMessage
    limit uint32
    size uint32
}

func (el *CacheLog) Dump(ctx *context.Context, in chan<- LogMessage, l *slog.Logger) {
    (*el).lock.Lock()
    mark := "CacheLod.Dump"
    l.Debug(fmt.Sprintf("%s | Locked, run dump...", mark))
    defer (*el).lock.Unlock()
    for i := uint32(0); i < (*el).size; i++ {
        select {
        case in<- (*el).records[i]:
            (*el).size--
        case <-(*ctx).Done():
            l.Debug(fmt.Sprintf("%s | Cancelled...", mark))
        }
    }
    (*el).records = nil
    l.Debug(fmt.Sprintf("%s | Dumped %v ...", mark, (*el).records))
}

func (el *CacheLog) LogEvicted(val string) error {
    (*el).lock.Lock()
    defer (*el).lock.Unlock()
    if (*el).size == (*el).limit && (*el).limit > 0 {
        return errors.New("Cache log overflow...")
    }
    (*el).records = append((*el).records, CacheLogMessage{Evicted, val})
    return nil
}

func (el *CacheLog) LogAdded(val string) error {
    (*el).lock.Lock()
    defer (*el).lock.Unlock()
    if (*el).size == (*el).limit && (*el).limit > 0 {
        return errors.New("Cache log overflow...")
    }
    (*el).records = append((*el).records, CacheLogMessage{Added, val})
    return nil
}

func NewCacheLog(limit uint32) *CacheLog {
    return &CacheLog{limit: limit, lock: sync.RWMutex{}}
}

type CacheServIntf interface {
    SetOne(msg *struct{}) error
    SetMany(msg *struct{}) error
    Get(key interface{}) error
    Listen(ch <-chan interface{})
    SetLogger(log interface{})
    MarkAdded(key string)
    MarkEvicted(key string)
    Run()
}

type AppCache struct {
    ctx *context.Context
    log *slog.Logger
    dump_it time.Duration
    income <-chan CacheItem
    errCh chan<- error
    c *cache.AppLRUCache
    evLog *CacheLog
}

func NewCacheService(
    ctx *context.Context,
    errch chan<- error,
    cache *cache.AppLRUCache,
    ) AppCache {
    return AppCache{
        ctx:            ctx,
        errCh:          errch,
        c:              cache,
        evLog:          NewCacheLog(2048),
    }
}

func (ca *AppCache) MarkEvicted(key string) {
    (*ca.evLog).LogEvicted(key)
}

func (ca *AppCache) MarkAdded(key string) {
    (*ca.evLog).LogAdded(key)
}

func (ca *AppCache) Listen(ch <-chan CacheItem) {
    (*ca).income = ch
}

func (ca *AppCache) SetLogger(log *slog.Logger) {
    (*ca).log = log
    (*ca.log).Debug("Configured logger in cache service...")
}

func (ca *AppCache) Run() {

    // cache swap service
    go func(c *AppCache) {
        var syncErr error
        mark := "AppCache.Run"
        (*ca).log.Debug(mark)
        for msg := range (*c).income {
            select {
            case <-(*c.ctx).Done():
                return
            default:
                switch msg.kind {
                case AddOne:
                    syncErr = (*c).SetOne(msg)
                case AddMany:
                    syncErr = (*c).SetMany(msg)
                default:
                    errMsg := fmt.Sprintf("%s: Unknown msg -> %s", mark, msg.kind)
                    syncErr = errors.New(errMsg)
                }
            }
            if syncErr != nil {
                errMsg := fmt.Errorf("%s: error %w", mark, syncErr)
                select {
                case (*c).errCh<- errMsg:
                case <-(*c.ctx).Done():
                    return
                }
            }
        }
    }(ca)
}

func (ca *AppCache) GetCacheSync(cb func(<-chan LogMessage)) func() {
    // evicted and added dump service
    return func() {
        dump_ch := make(chan LogMessage)
        call := cb
        go func(c *AppCache, dump chan LogMessage, cb func(<-chan LogMessage)) {
            mark := "AppCache.DumpBackground"
            (*c).log.Debug(mark)

            ticker := time.NewTicker((*c).dump_it)
            defer ticker.Stop()
            for {
                select {
                case <- ticker.C:
                    (*c).log.Debug(fmt.Sprintf("%s | Run cache dump...", mark))
                    go (*c).evLog.Dump(c.ctx, dump, c.log)
                    cb(dump)
                case <-(*c.ctx).Done():
                    return
                }
            }
        }(ca, dump_ch, call)
    }
}

func (ca *AppCache) SetOne(item CacheItem) error {
    mark := "AppCache.SetOne"
    if (*ca).c == nil {
        msg := fmt.Sprintf("%s, error Cache not set.", mark)
        return errors.New(msg)
    }
    msg := item.payload.(Order)
    _, err := (*ca).c.Setex(msg.Id(), msg.Payload(), (*ca).c.ExpT)
    if err != nil {
        return fmt.Errorf("%s, error %w", mark, err)
    }
    return nil
}

func (ca *AppCache) SetMany(item CacheItem) error {
    mark := "AppCache.SetMany"
    if (*ca).c == nil {
        msg := fmt.Sprintf("%s, error Cache not set.", mark)
        return errors.New(msg)
    }
    orders := item.payload.(Orders)
    for _, order := range orders.GetItems() {
        (*ca.log).Debug(fmt.Sprintf("%s | Order = %+v", mark, order))
        _, err := (*ca).c.Setex(order.oid, order.payload, (*ca).c.ExpT)
        if err != nil {
            return fmt.Errorf("%s: error %w", mark, err)
        }
    }
    return nil
}

func (ca *AppCache) Get(key string) (Order, error) {
    mark := "AppCache.Get"
    var err error
    if (*ca).c == nil {
        msg := fmt.Sprintf("%s, error Cache not set.", mark)
        return Order{}, errors.New(msg)
    }
    ordr, err := (*ca).Get(key)
    if err != nil {
        // if no key, we have to fetch them from db
        if (*ca).c.On_load == nil {
            return Order{}, fmt.Errorf("%s, error %w", mark, err)
        }
        from_db := (*ca).c.On_load(key)
        if from_db == nil {
            return Order{}, fmt.Errorf("%s, error %w", mark, err)
        }
        return from_db.(Order), nil
    }
    return ordr, nil
}
