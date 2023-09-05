package main

import (
    "fmt"
    "os"
    "log/slog"
    "errors"
)

const (
    LocalEnv string = "local"
    DevEnv string = "dev"
    ProdEnv string = "prod"
)

// setup new logger
func SetupLogger(env string) (*slog.Logger, error) {
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
        return logger, errors.New(fmt.Sprintf("Invalid env: %s", env))
    }
    return logger, nil
}
