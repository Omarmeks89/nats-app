package main

import (
    "os"
    "fmt"
    "nats_app/internal/config"
    "nats_app/internal/storage/psql"
    api "nats_app/internal/http-server/handlers/api"

    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
    "github.com/go-playground/validator/v10"
)

const (
    AppConfPathKey string = "N_APP_CONFIG"
)

var RequestValidator = validator.New()

// mock main()
func main () {
    conf, err := config.MustBuildConfig(AppConfPathKey)
    if err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
    logger, err := SetupLogger(conf.Env)
    if err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
    // fmt.Printf("builded: %+v\n", conf)
    logger.Info("Bootstrap...")
    logger.Debug("Debug messages enabled...")
    var dbAdapter psql.PostgreDB = psql.PostgreDB{}
    router := chi.NewRouter()
    router.Use(middleware.RequestID)
    router.Use(middleware.Recoverer)
    router.Get("/url", api.GetOrder(logger, RequestValidator, &dbAdapter))
}
