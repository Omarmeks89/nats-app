package main

import (
    "os"
    "log/slog"
    "log"
)

const (
    LocalEnv string = "local"
    DevEnv string = "dev"
    ProdEnv string = "prod"
)

// setup new logger
func SetupLogger(env string) *slog.Logger {
    var logger *slog.Logger
    switch env {
    case LocalEnv:
        logger = slog.New(
            slog.NewTextHandler(
                os.Stdout,
                &slog.HandlerOptions{Level: slog.LevelDebug},
            ),
        )
    case DevEnv:
        logger = slog.New(
            slog.NewJSONHandler(
                os.Stdout,
                &slog.HandlerOptions{Level: slog.LevelDebug},
            ),
        )
    case ProdEnv:
        logger = slog.New(
            slog.NewJSONHandler(
                os.Stdout,
                &slog.HandlerOptions{Level: slog.LevelInfo},
            ),
        )
    default:
        log.Fatal("Env mode not allowed. Use: <local>, <dev> or <prod>.")
    }
    return logger
}
