package main

import (
    "fmt"
    "context"
    "log/slog"

    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
    "github.com/go-playground/validator/v10"

    "nats_app/internal/config"
    "nats_app/internal/storage/psql"
    "nats_app/internal/storage/cache"
    "nats_app/internal/nats_client"
    "nats_app/internal/services"
    api "nats_app/internal/http-server/handlers/api"
)

const (
    AppConfPathKey string = "N_APP_CONFIG"
)

var (
    RequestValidator = validator.New()
    GetErrChan func() <-chan error
    Cancel func()
    Ctx context.Context
    dbAdapter *psql.PostgreDB
    Conf *config.AppConfig
    Cache services.AppCache
    Storage services.AppStorage
    Consumer nats_client.AppConsumer
    LogStateSync func()
    logger *slog.Logger
)

// bootstrap
func init() {
    ErrCh := make(chan error)
    GetErrChan = func() <-chan error {return ErrCh}
    // we will use logger.Fatal
    Conf = config.MustBuildConfig(AppConfPathKey)
    logger = SetupLogger(Conf.Env)
    Ctx, Cancel = context.WithTimeout(
        context.Background(),
        Conf.HTTPConf.ResponseTimeout,
    )
    logger.Info("Bootstrap...")
    dbAdapter := psql.NewDB(Ctx, &Conf.DBConf)

    logger.Debug("Connection to db created...")
    psql.Ping(dbAdapter)

    logger.Debug("DB answered...")
    PoolSize := Conf.StoragePoolSize
    Storage := services.NewStorage(&Ctx, *dbAdapter, PoolSize, ErrCh)
    Storage.SetLogger(logger)

    Consumer := nats_client.NewStanConsumer(&Ctx, ErrCh, &Conf.StanConf)
    Consumer.SetStorageOnCallback(Storage)
    Consumer.SetLogger(logger)

    // setup Cache
    LRUCache := cache.NewLRUCache(&Conf.CacheConf).
        OnEvict(
            func(key string, val []byte) {
                Cache.MarkEvicted(key)
                return
            },
        ).
        OnAdd(
            func(key string, val []byte) {
                Cache.MarkAdded(key)
            },
        ).
        OnLoad(func(key string) interface{} {
            out := Storage.FetchOrder(key)
            for ord := range out {
                return ord
            }
            return nil
        }).
        Build()
    Cache = services.NewCacheService(&Ctx, ErrCh, LRUCache)
    Cache.SetLogger(logger)
    LogStateSync = Cache.GetCacheSync(
        func(c <-chan services.LogMessage) {
            Storage.MarkDumped(c)
            return
        },
    )

    logger.Debug("Setup for start...")
    data_chan := Storage.GetChannel()
    Cache.Listen(data_chan)

    return
}

// on shutdown
func on_shutdown(cancel func()) {
    //...
    logger.Info("Stopping services...")
    cancel()

    logger.Info("Disconnection...")
    err := Consumer.Disconnect()
    if err != nil {
        logger.Error(fmt.Sprintf("Error on disconnect: %s", err.Error()))
    }
    Storage.Disconnect()
    logger.Info("Done...")
    return
}

func main () {

    var intError error
    Errors := GetErrChan()
    defer on_shutdown(Cancel)

    logger.Debug("Run services...")
    Cache.Run()
    LogStateSync()

    logger.Debug("Checking start mode...")
    if F, Crashed := utils.CheckCrashes(); Crashed != nil {
        logger.Debug("Start in rebuild mode...")
        // now we send errors to errChannel
        Storage.RestoreCache(Conf.RestoreRecordsLimit)
        LastTimestamp, _ := utils.GetTimestamp(F)
        err := Consumer.RunFromTimestamp(LastTimestamp)
        if err != nil {
            logger.Error(fmt.Sprintf("Error on consumer start: %s", err.Error()))
            return
        }
    } else {
        logger.Debug("Start in normal mode...")
        _, FileCreationErr := utils.FixStartup()
        if FileCreationErr != nil {
            logger.Error(FileCreationErr)
            return
        } else {
            err := Consumer.Run()
            if err != nil {
                logger.Error(fmt.Sprintf("Error on consumer start: %s", err.Error()))
                return
            }
        }
    }
    utils.RunTimestampWriting(&Conf.TSUpdateInterval, Errors)

    logger.Debug("Setup routers...")
    router := chi.NewRouter()
    router.Use(middleware.RequestID)
    router.Use(middleware.Recoverer)
    router.Get("/orders", api.GetOrder(logger, RequestValidator, &Cache))

    // main loop
    for {
        logger.Info("Running...")
        select {
        case intError = <-Errors:
            logger.Error(fmt.Sprintf("main | Error: %s", intError.Error()))
            var DBCritical *app_errors.DBConnectionLost
            switch intError {
                case DBCritical: 
                    if _, err := Storage.TestConnection(); err != nil {
                        logger.Error("DB connection lost...")
                        return
                    }
                    logger.Info("DB connection found...")
            default:
                break
            }
        }
    }
    logger.Info("Shutting down...")
    return
}
