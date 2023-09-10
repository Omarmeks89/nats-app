package storage

import (
    "log/slog"

    "github.com/jackc/pgx/v5"

    "nats_app/internal/storage/psql"
)

type DBAdapter interface {
    Test() error
    SetLogger(l *slog.Logger)
    BeginTx() (psql.Transaction, error)
    Save(q string, args ...any) (func(), error)
    FetchOne(q string, args ...any) *psql.SingleOpFuture
    FetchMany(q string, args ...any) (pgx.Rows, func(), error)
    Disconnect()
}
