package services

import (
    "fmt"
    "log/slog"
    "context"
    "time"
    "os"

    "github.com/jackc/pgx/v5"

    "nats_app/internal/storage"
    "nats_app/internal/storage/psql"
)

type Token uint8

type Order struct {
    Oid string
    // payload -> json
    Payload *[]byte
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
    GetPayload() interface{}
}


func (o Order) Id() string {
    return o.Oid
}

func (o Order) GetPayload() *[]byte {
    return o.Payload
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
    ctx context.Context
    db storage.DBAdapter
    log slog.Logger
    wPool chan Token
    outCh chan CacheItem
    errCh chan<- error
}

// biuld new AppStorage
func NewStorage(
    ctx context.Context,
    dba storage.DBAdapter,
    pool_size int,
    errch chan<- error,
    ) AppStorage {

    wpool := make(chan Token, pool_size)
    for i := 0; i < pool_size; i++ {
        wpool<- Token(i)
    }
    outCh := make(chan CacheItem)
    return AppStorage{
        ctx:            ctx,
        db:             dba,
        wPool:          wpool,
        outCh:          outCh,
        errCh:          errch,
        log:            *slog.New(
                            slog.NewTextHandler(
                                 os.Stdout,
                                 &slog.HandlerOptions{Level: slog.LevelDebug},
                            ),
                        ),
    }
}

func (srv AppStorage) SetLogger(l slog.Logger) {
    srv.log.Debug("AppStorage logger setup...")
}

// return channel for items that will 
// fetch data from db
func (srv AppStorage) GetChannel() <-chan CacheItem {
    return srv.outCh
}

// break connection with db
func (srv AppStorage) Disconnect() {
    srv.db.Disconnect()
}

// restore cache if we felt down
func (srv AppStorage) RestoreCache(records int, timeout time.Duration) func() {
    //...
    mark := "AppStorage.RestoreCache"
    tmpCtx, cancel := context.WithTimeout(srv.ctx, timeout)

    srv.log.Debug(mark)
    go func(s AppStorage, ctx context.Context, limit int) {

        var t Token
        defer func(srv AppStorage, t Token) {s.wPool<- t}(s, t)
        var offset int
        var orders []Order
        var cancel func()
        query := "SELECT oid, raw_ord FROM orders WHERE evict=$1 ORDER BY seq_idx DESC LIMIT $2 OFFSET $3"
        batch := int(limit / 10)
        for i := 0; i < 10; i++ {
            select {
            case <-ctx.Done():
                return
            case t = <-s.wPool:
                rows, cancel, err := s.db.FetchMany(query, Evicted, batch, offset)
                if err != nil {
                    cancel()
                    select {
                    case <-ctx.Done():
                        return
                    case s.errCh<- err:
                        return
                    }
                }
                Rows := rows.(pgx.Rows)
                for Rows.Next() {
                    var ord Order
                    if err := Rows.Scan(&ord.Oid, &ord.Payload); err != nil {
                        s.log.Error(fmt.Sprintf("Can`t parse data... | %+v", err))
                        continue
                    }
                    orders = append(orders, ord)
                }
                offset += batch
                ords := Orders{items: orders}
                cItem := CacheItem{kind: AddMany, payload: ords}
                select {
                case <-ctx.Done():
                    cancel()
                    return
                case s.outCh<- cItem:
                    orders = nil
                }
            }
            // we return tokem each iteration
            select {
            case s.wPool<- t:
            case <-ctx.Done():
                cancel()
                return
            }
        }

        s.log.Debug("Cache restoration finished...")
    }(srv, tmpCtx, records)

    return cancel
}

func (srv AppStorage) TestConnection() (bool, error) {
    //...
    srv.log.Debug("Ping DB...")
    mark := "AppStorage.TestConnection"
    err := srv.db.Test()
    if err != nil {
        srv.log.Error(fmt.Sprintf("%s | Error: %+v", mark, err))
        return false, fmt.Errorf("%s | Error: %w", mark, err)
    }
    return false, nil
}

func (srv AppStorage) Convert(id uint64, oid string, data *[]byte) *NatsMsg {
    o := Order{oid, data}
    return &NatsMsg{id, o}
}

func (srv AppStorage) SaveOrder(nm *NatsMsg) {
    
    o := Order{Oid: (*nm).Oid, Payload: nm.Payload}
    query := "INSERT INTO orders (oid, raw_ord) VALUES ($1, $2)"

    go func(s AppStorage, query *string, o *Order) {
        var t Token
        mark := "AppStorage.SaveOrder"
        defer func(srv AppStorage, t Token) {s.wPool<- t}(s, t)
        select {
        case t = <-s.wPool:
            Cancel, DBErr := s.db.Save(*query, o.Oid, *o.Payload)
            if DBErr != nil {
                Cancel()
                DBErr = fmt.Errorf("%s: %w", mark, DBErr)
                errText := fmt.Sprintf("%s [GORO] | Error %s", mark, DBErr.Error())
                s.log.Error(errText)
                select {
                case s.errCh<- DBErr:
                    return
                }
            }
            cItem := CacheItem{AddOne, *o}
            select {
            case <-s.ctx.Done():
            case s.outCh<- cItem:
            }
        case <-s.ctx.Done():
        }
        return
    }(srv, &query, &o)

    return
}

func (srv AppStorage) FetchOrder(oid string) Order {
    // make query

    query := "SELECT oid, raw_ord FROM orders WHERE oid=$1"
    mark := "AppStorage.FetchOrder"

    var t Token
    var ord Order
    defer func(srv AppStorage, t Token) {srv.wPool<- t}(srv, t)
    select {
    case t = <-srv.wPool:
        fetched := srv.db.FetchOne(query, oid)
        DBErr := fetched.ParseInto(&ord.Oid, &ord.Payload)
        if DBErr != nil {
            DBErr = fmt.Errorf("%s: %w", mark, DBErr)
            srv.log.Debug(fmt.Sprintf("Error... %s", DBErr.Error()))
            select {
            case srv.errCh<- DBErr:
            case <-srv.ctx.Done():
            }
        }
    }
    srv.log.Debug(fmt.Sprintf("%s | Order [%s] found", mark, oid))
    return ord
}

func (srv AppStorage) MarkDumped(ch <-chan LogMessage, ca func()) {
    // make queries from str array for trans.
    // it will be called from <GatCacheSync>

    query := "UPDATE orders SET evict = $2 WHERE oid = $1"

    var t Token
    mark := "AppStorage.MarkDumpedBG"
    srv.log.Debug(fmt.Sprintf("%s | Started... | Pool %+v, %d", mark, srv.wPool, len(srv.wPool)))
    defer ca()
    defer func(s AppStorage, t Token) {srv.wPool<- t}(srv, t)
    // writer will close channel
    // or caller close it when ctx will be Done().
    select {
    case <-srv.ctx.Done():
        return
    case t = <-srv.wPool:
        // open transaction
        var Trans psql.Transaction
        var TrError error
        Trans, TrError = srv.db.BeginTx()
        if TrError != nil {
            select {
            case <-srv.ctx.Done():
                return
            case srv.errCh<- fmt.Errorf("%s | Error %w", mark, TrError):
                return
            }
        }
        for msg := range ch {
            switch msg.OpCode() {
            case Evicted:
                Trans.AddQuery(query, string(msg.Payload()), Evicted)
            case Added:
                Trans.AddQuery(query, string(msg.Payload()), Added)
            default:
                srv.log.Error(fmt.Sprintf("%s | Unknown op = %d", mark, msg.OpCode()))
                Trans.Rollback()
                select {
                case <-srv.ctx.Done():
                case srv.errCh<- fmt.Errorf("%s | Unknown opcode", mark):
                    return
                }
            }
        }
        // close transaction
        TrError = Trans.RunTx()
        if TrError != nil {
            srv.log.Debug(fmt.Sprintf("%s | Transaction Rolled back...", mark))
            Trans.Rollback()
            select {
            case <-srv.ctx.Done():
            case srv.errCh<- fmt.Errorf("%s | Error %w", mark, TrError):
            }
        }
        Trans.Commit()
    }
    return
}
