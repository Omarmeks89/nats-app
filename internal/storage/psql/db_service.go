package psql

import (
    "context"
    "fmt"
    "errors"
    "github.com/jackc/pgx/v5/pgxpool"
    "nats_app/internal/config"
    
)

// postgres adapter implementation
type PostgreDB struct {
    Ctx context.Context
    pool *pgxpool.Pool
}

// method, that check db conn alive
func (psql *PostgreDB) Test() error {
    if NoPing := (*psql.pool).Ping((*psql).Ctx); NoPing != nil {
        return fmt.Errorf("Database is not responding... ERR: %w", NoPing)
    }
    return nil
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
func NewDB(ctx context.Context, s *config.DBEngineConf) (*PostgreDB, error) {
    DbUrl, CredErr := buildDbUrl(s)
    if CredErr != nil {
        return &PostgreDB{}, fmt.Errorf("Can`t create connection pool. Error: %w", CredErr)
    }
    for i := 0; i < (*s).ConnRetry; i++ {
        pool, connErr := pgxpool.New(ctx, DbUrl)
        if connErr != nil {
            continue
        } else {
            return &PostgreDB{ctx, pool}, nil
        }
    }
    return &PostgreDB{}, errors.New("Can`t connect to DB")
}
