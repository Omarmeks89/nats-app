package psql

import (
    "context"
    "time"
    "github.com/jackc/pgx/v5"
)

type DBSettings struct {
    //...
}

type DBErr struct {
    msg string
    errType error
}

func (e DBErr) Error() string {
    return e.msg
}

func (e DBErr) Unwrap() error {
    return e.errType
}

// fst impl of dbAdapter interface
type DBAdapter interface {
    Connect(ctx context.Context, s DBSettings) (bool, error)
    Begin() error
    Commit() error
    Rollback() error
    Insert(query string) error
    Select(query string) (any, error)
    SelectMany(query string) ([]any, error)
}

type PostgreDB struct {
    //...
    Ctx context.Context
    Host string
    Port string
    DBName string
    Passwd string
    User string
    MaxPool uint
    Timeout time.Duration
}

func (psql *PostgreDB) Connect(ctx context.Context, s DBSettings) (bool, DBErr) {
    //...
}

func (psql *PostgreDB) Begin() DBErr {
    //...
}

func (psql *PostgreDB) Commit() DBErr {
    //...
}

func (psql *PostgreDB) Rollback() DBErr {
    //...
}

func (psql *PostgreDB) Insert(query string) error {
    //...
}

func (psql *PostgreDB) Select(query string) (any, error) {
    //...
}

func (psql *PostgreDB) SelectMany(query string) ([]any, error) {
    //...
}

// init new db adapter
// ctx -> current Context
// s -> db settings
func NewDB(ctx context.Context, s any) (*DBAdapter, error) {
    //...
}
