package psql

import (
    "context"
    "time"
    // "github.com/jackc/pgx/v5"
    
)

// postgres adapter implementation
type PostgreDB struct {
    Ctx context.Context
    MaxPool uint
    Timeout time.Duration
    pool any
}

func (psql *PostgreDB) Connect(ctx context.Context) (bool, error) {
    //...
    return false, nil
}

func (psql *PostgreDB) Begin() error {
    //...
    return nil
}

func (psql *PostgreDB) Commit() error {
    //...
    return nil
}

func (psql *PostgreDB) Rollback() error {
    //...
    return nil
}

func (psql *PostgreDB) Save(query string) error {
    //...
    return nil
}

func (psql *PostgreDB) FetchOne(query string) (any, error) {
    //...
    return false, nil
}

func (psql *PostgreDB) FetchMany(query string) (any, error) {
    //...
    return false, nil
}

// init new db adapter
// ctx -> current Context
// s -> db settings
func NewDB(ctx context.Context, s any) (*PostgreDB, error) {
    //...
    return &PostgreDB{}, nil
}
