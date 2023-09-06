package storage

import (
    "errors"
)

var (
    DBSetupError = errors.New("invalid db setup parameters")
    TransactionError = errors.New("Transaction failed")
)

// fst impl of dbAdapter interface
type DBAdapter interface {
    Test() error
    Begin() error
    Commit() error
    Rollback() error
    Save(query string) error
    FetchOne(query string) (any, error)
    FetchMany(query string) (any, error)
    Close()
}
