package psql

import (
    "context"
    "os"
    "fmt"
    "time"
    "errors"
    "log/slog"
    "log"
    
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/jackc/pgx/v5"

    "nats_app/internal/config"
)

type OpFuture interface {
    ParseInto(args ...any) error
}

type SingleOpFuture struct {
    row pgx.Row
    cancel_f func()
}

func (sof *SingleOpFuture) ParseInto(args ...any) error { 
    //...
    var err error
    defer (*sof).cleanup()
    mark := "PostgreDB.SingleOpFuture.Scan"
    err = (*sof).row.Scan(args...)
    if err != nil {
        err = fmt.Errorf("%s | Scan error %w", mark, err)
    }
    return err
}

func (sof *SingleOpFuture) cleanup() {
    //...
    (*sof).row = nil
    defer (*sof).cancel_f()
}

// postgres adapter implementation
type PostgreDB struct {
    Ctx context.Context
    pool *pgxpool.Pool
    log *slog.Logger
    timeout time.Duration
}

type Transaction struct {
    Tx pgx.Tx
    Batch *pgx.Batch
    Ctx context.Context
    cancel_f func()
    log *slog.Logger
}

// method, that check db conn alive
func (psql PostgreDB) Test() error {
    if NoPing := (*psql.pool).Ping(psql.Ctx); NoPing != nil {
        return fmt.Errorf("Database is not responding... ERR: %w", NoPing)
    }
    return nil
}

func (psql PostgreDB) SetLogger(l *slog.Logger) {
    psql.log.Debug("Logger set in DB Adapter...")
}

func (psql PostgreDB) BeginTx() (Transaction, error) {
    // we set pgx.Tx inside and send
    // error if occured
    psql.log.Debug("Transaction begin...")
    mark := "PostgreDB.BeginTx"
    timeCtx, cancel := context.WithTimeout(psql.Ctx, psql.timeout)
    // pool use BeginTx under the hood
    Tx, txErr := psql.pool.Begin(timeCtx)
    if txErr != nil {
        defer cancel()
        return Transaction{}, fmt.Errorf("%s | Error %w", mark, txErr)
    }
    return Transaction{
        Tx:             Tx,
        Batch:          &pgx.Batch{},
        Ctx:            timeCtx,
        cancel_f:       cancel,
        log:            slog.New(slog.NewTextHandler(os.Stderr, nil)),
    }, nil
}

func (tr *Transaction) AddQuery(q string, args ...any) error {
    if tr.Batch == nil {
        return errors.New("AQ | No opened transactions...")
    }
    a := fmt.Sprintf("args => %v, %+v", args, args)
    tr.log.Info(a)
    tr.Batch.Queue(q, args...)
    return nil
}

func (tr *Transaction) RunTx() error {
    var Err error
    tr.log.Info("Run transaction (SendBatch)...")
    mark := "PostgreDB.RunTx"
    if tr.Tx == nil {
        return errors.New("RTX | No opened transactions...")
    }
    br := tr.Tx.SendBatch(tr.Ctx, tr.Batch)
    if _, Err = br.Exec(); Err != nil {
        Err = fmt.Errorf("%s | Error on Batch.Exec() %w", mark, Err)
    }
    CloseErr := br.Close()
    if CloseErr != nil {
        Err = fmt.Errorf("%s | Error on Batch.Close() %w", mark, Err)
    }
    return Err
}

func (tr *Transaction) Commit() error {
    //...
    var Err error
    tr.log.Info("Commit...")
    mark := "PostgreDB.Commit"
    if tr.Tx == nil {
        return errors.New("CM | No opened transactions...")
    }
    if Err = tr.Tx.Commit(tr.Ctx); Err != nil {
        Err = fmt.Errorf("%s | Commit failed. Error on RB %w", mark, Err)
    }
    defer tr.cancel_f()
    return Err
}

func (tr *Transaction) Rollback() error {
    //...
    var Err error
    tr.log.Info("Rollback...")
    mark := "PostgreDB.Rollback"
    if tr.Tx == nil {
        return errors.New("RB | No opened transactions...")
    }
    if Err = tr.Tx.Rollback(tr.Ctx); Err != nil {
        Err = fmt.Errorf("%s | Error on rollback %w", mark, Err)
    }
    defer tr.cancel_f()
    return Err
}

func (psql PostgreDB) Save(q string, args ...any) (func(), error) {
    //...
    mark := "PostgreDB.Save"
    tempCtx, cancel := context.WithTimeout(psql.Ctx, psql.timeout)
    _, err := psql.pool.Exec(tempCtx, q, args...)
    if err != nil {
        psql.log.Error("Error in Save")
        return cancel, fmt.Errorf("%s | Error %w", mark, err)
    }
    return cancel, nil
}

func (psql PostgreDB) FetchOne(q string, args ...any) *SingleOpFuture {
    //...
    tempCtx, cancel := context.WithTimeout(psql.Ctx, psql.timeout)
    obj := psql.pool.QueryRow(tempCtx, q, args...)
    return &SingleOpFuture{row: obj, cancel_f: cancel}
}

func (psql PostgreDB) FetchMany(q string, args ...any) (pgx.Rows, func(), error) {
    //...
    mark := "PostgreDB.FetchMany"
    var rows pgx.Rows
    var err error
    tempCtx, cancel := context.WithTimeout(psql.Ctx, psql.timeout)
    shutdown_handler := func() {
        defer cancel()
        defer rows.Close()
    }
    rows, err = psql.pool.Query(tempCtx, q, args...)
    if err != nil {
        psql.log.Error("Error in FetchMany")
        defer cancel()
        return rows, func(){}, fmt.Errorf("%s | Error %w", mark, err)
    }
    return rows, shutdown_handler, nil
}

func (psql PostgreDB) Disconnect() {
    //...
    psql.pool.Close()
    psql.log.Info("DB Pool closed...")
}


func Ping(db *PostgreDB) {
    //...
    if err := (*db).Test(); err != nil {
        log.Fatal("No <PONG> from DB...")
    }
}

// validate db tokens
func validateDbUrlTokens(t ...string) error {
    for _, token := range t {
        if token == "" {
            return errors.New("One of url tokens is empty")
        }
    }
    return nil
}

// building DB url from config
func buildDbUrl(s *config.DBEngineConf) (string, error) {
    if err := validateDbUrlTokens(
        (*s).Driver,
        (*s).Db_admin,
        (*s).Passwd,
        (*s).Host,
        (*s).Port,
        (*s).DBName,
    ); err != nil {
        return "", errors.New("Invalid db credentials")
    }
    return fmt.Sprintf(
        "%s://%s:%s@%s:%s/%s",
        (*s).Driver,
        (*s).Db_admin,
        (*s).Passwd,
        (*s).Host,
        (*s).Port,
        (*s).DBName,
    ), nil
}

// init new db adapter
// ctx -> current Context
// s -> db settings
func NewDB(ctx context.Context, s *config.DBEngineConf) *PostgreDB {
    DbUrl, CredErr := buildDbUrl(s)
    if CredErr != nil {
        log.Fatal("Can`t create connection pool.")
    }
    for i := 0; i < (*s).ConnRetry; i++ {
        pool, connErr := pgxpool.New(ctx, DbUrl)
        if connErr != nil {
            continue
        } else {
            return &PostgreDB{Ctx: ctx, pool: pool, timeout: (*s).Timeout, log: slog.New(slog.NewTextHandler(os.Stdout, nil))}
        }
    }
    log.Fatal("Can`t set connection to DB.")
    return &PostgreDB{}
}
