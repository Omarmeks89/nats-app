package services

import (
    "fmt"
    "log/slog"
    "context"

    "nats_app/internal/storage"
)

type Token uint8

type Order struct {
    oid string
    // payload -> json
    payload *[]byte
}

type Orders struct {
    // many orders, when we restore cache
    items []Order
}

func (ords *Orders) GetItems() []Order {
    return (*ords).items
}

// represent msg from NATS
type NatsMsg struct {
    MsgId uint64
    Order
}

type CacheItem struct {
    kind string
    payload interface{}
}

type CustOrder interface {
    Id() interface{}
    Payload() interface{}
}


func (o Order) Id() string {
    return o.oid
}

func (o Order) Payload() *[]byte {
    return o.payload
}

func NewOrder(id string, payload *[]byte) Order {
    return Order{id, payload}
}

type StorageInterface interface {
    SetLogger(l *slog.Logger)
    GetChannel() <-chan interface{}
    GetLoader() func(interface{}, interface{}) error
    RestoreCache(records int)
    Disconnect()
    TestConnection() (bool, error)
    SaveOrder(mn interface{})
    Convert() interface{}
    FetchOrder(oid string) (interface{}, error)
    MarkDumped(<-chan interface{})
}

type AppStorage struct {
    ctx *context.Context
    db storage.DBAdapter
    log *slog.Logger
    wPool chan Token
    outCh chan CacheItem
    errCh chan<- error
}

// biuld new AppStorage
func NewStorage(
    ctx *context.Context,
    dba storage.DBAdapter,
    pool_size int,
    errch chan<- error,
    ) *AppStorage {

    wpool := make(chan Token, pool_size)
    outCh := make(chan CacheItem)
    return &AppStorage{
        ctx:            ctx,
        db:             dba,
        wPool:          wpool,
        outCh:          outCh,
        errCh:          errch,
    }
}

func (srv *AppStorage) SetLogger(l *slog.Logger) {
    (*srv).log = l
}

// return channel for items that will 
// fetch data from db
func (srv *AppStorage) GetChannel() <-chan CacheItem {
    return (*srv).outCh
}

// break connection with db
func (srv *AppStorage) Disconnect() {
    // NotImplemented
}

// restore cache if we felt down
func (srv *AppStorage) RestoreCache(records int) {
    //...
}

func (srv *AppStorage) TestConnection() (bool, error) {
    //...
    srv.log.Debug("Ping DB...")
    mark := "AppStorage.TestConnection"
    err := (*srv).db.Test()
    if err != nil {
        srv.log.Error(fmt.Sprintf("%s | Error: %+v", mark, err))
        return false, fmt.Errorf("%s | Error: %w", mark, err)
    }
    return false, nil
}

func (srv *AppStorage) Convert(id uint64, oid string, data *[]byte) *NatsMsg {
    mark := "AppStorage.Convert"
    o := Order{oid, data}
    srv.log.Debug(fmt.Sprintf("%s | Order = %+v", mark, o))
    return &NatsMsg{id, o}
}

func (srv *AppStorage) SaveOrder(nm *NatsMsg) {
    
    o := Order{oid: (*nm).oid, payload: nm.payload}
    query := fmt.Sprintf("SELECT * FROM orders; %+v", o)

    go func(s *AppStorage, query *string, o *Order) {
        var t Token
        defer func(srv *AppStorage, t Token) {(*s).wPool<- t}(s, t)
        select {
        case t = <-(*s).wPool:
            DBErr := (*s).db.Save(*query)
            if DBErr != nil {
                mark := "AppStorage.SaveOrder"
                DBErr = fmt.Errorf("%s: %w", mark, DBErr)
                select {
                case (*s).errCh<- DBErr:
                }
            }
            cItem := CacheItem{AddOne, o}
            select {
            case <-(*s.ctx).Done():
                (*s).db.Close()
            case (*s).outCh<- cItem:
            }
        case <-(*s.ctx).Done():
            (*s).db.Close()
        }
        return
    }(srv, &query, &o)

    return
}

func (srv *AppStorage) FetchOrder(oid string) <-chan Order {
    // make query
    out := make(chan Order)
    query := fmt.Sprintf("SELECT payload FROM orders WHERE oid == %s", oid)

    go func(s *AppStorage, oid, query *string) {
        var t Token
        defer func(srv *AppStorage, t Token) {(*s).wPool<- t}(s, t)
        defer close(out)
        select {
        case t = <-(*s).wPool:
            ordr, DBErr := (*s).db.FetchOne(*query)
            if DBErr != nil {
                mark := "AppStorage.FetchOrder"
                DBErr = fmt.Errorf("%s: %w", mark, DBErr)
                select {
                case (*s).errCh<- DBErr:
                }
            }
            select {
            case <-(*s.ctx).Done():
                (*s).db.Close()
            case out<- Order{*oid, &ordr}:
            }
        case <-(*s.ctx).Done():
            (*s).db.Close()
        }
        return
    }(srv, &oid, &query)

    return out
}

func (srv *AppStorage) MarkDumped(ch <-chan LogMessage) {
    // make queries from str array for trans.
    go func(s *AppStorage, ch <-chan LogMessage) {
        var t Token
        mark := "AppStorage.MarkDumpedBG"
        (*s).log.Debug(fmt.Sprintf("%s | Started...", mark))
        defer func(s *AppStorage, t Token) {(*s).wPool<- t}(s, t)
        select {
        case msg := <-ch:
            //... handle
            select {
            case <-(*s.ctx).Done():
                return
            case t = <-(*s).wPool:
                // start work
                _ = t
                _ = msg
            }
        case <-(*s.ctx).Done():
            return
        }
    }(srv, ch)

    return
}
