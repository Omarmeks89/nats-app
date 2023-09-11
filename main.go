package main

import (
    "fmt"
    "context"
    "log/slog"
    "time"
    "net/http"

    "github.com/go-chi/chi/v5"
    "github.com/go-chi/cors"
    "github.com/go-chi/chi/v5/middleware"
    "github.com/go-playground/validator/v10"

    "nats_app/internal/config"
    "nats_app/internal/storage/psql"
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
    logger slog.Logger
)

// bootstrap
func init() {
    ErrCh := make(chan error)
    GetErrChan = func() <-chan error {return ErrCh}
    // we will use logger.Fatal
    Conf = config.MustBuildConfig(AppConfPathKey)
    logger = SetupLogger(Conf.Env)
    Ctx, Cancel = context.WithCancel(
        context.Background(),
    )
    logger.Info("Bootstrap...")
    dbAdapter = psql.NewDB(Ctx, &Conf.DBConf)
    dbAdapter.SetLogger(&logger)

    logger.Debug("Connection to db created...")
    psql.Ping(dbAdapter)

    logger.Debug("DB answered...")
    PoolSize := Conf.StoragePoolSize
    Storage = services.NewStorage(Ctx, *dbAdapter, PoolSize, ErrCh)
    Storage.SetLogger(logger)

    Consumer = nats_client.NewStanConsumer(Ctx, ErrCh, &Conf.StanConf)
    Consumer.SetStorageOnCallback(&Storage)
    Consumer.SetLogger(&logger)

    // setup Cache
    LRUCache := services.NewLRUCache(&Conf.CacheConf).
        OnEvict(
            func(key string, val *[]byte) {
                Cache.MarkEvicted(key)
                return
            },
        ).
        OnAdd(
            func(key string, val *[]byte) {
                Cache.MarkAdded(key)
            },
        ).
        OnLoad(func(key string) services.Order {
            return Storage.FetchOrder(key)
        }).
        Build()
    Cache = services.NewCacheService(&Ctx, ErrCh, LRUCache)
    Cache.SetLogger(&logger)

    // we will call this func from main 
    // to sync cache with db
    LogStateSync = Cache.GetCacheSync(
        Conf.TSUpdateInterval,
        func(c <-chan services.LogMessage, ca func()) {
            Storage.MarkDumped(c, ca)
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
    defer cancel()

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

    logger.Debug("Checking start mode...")
    if Crashed := services.CheckCrashed(); Crashed {
        logger.Debug("Start in rebuild mode...")
        // now we send errors to errChannel
        Storage.RestoreCache(Conf.RestoreRecordsLimit, Conf.TSUpdateInterval)
        LastTimestamp, TSErr:= services.GetPreviousTS(&logger)
        if TSErr != nil {
            logger.Error(fmt.Sprintf("Error %s", TSErr.Error()))
            return
        }
        err := Consumer.RunFromTimestamp(LastTimestamp)
        if err != nil {
            logger.Error(fmt.Sprintf("Error on consumer start: %s", err.Error()))
            return
        }
    } else {
        logger.Debug("Start in normal mode...")
        FileCreationErr := services.MakeNewTSFile(&logger)
        if FileCreationErr != nil {
            logger.Error(fmt.Sprintf("Error: %+v", FileCreationErr))
            return
        } else {
            err := Consumer.Run()
            if err != nil {
                logger.Error(fmt.Sprintf("Error on consumer start: %s", err.Error()))
                return
            }
        }
    }

    logger.Debug("Setup routers...")
    router := chi.NewRouter()
    cors := cors.New(cors.Options{
	AllowedOrigins:   []string{"*"},
	AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
	AllowedHeaders:   []string{"X-PINGOTHER", "Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
	AllowCredentials: true,
	MaxAge:           300,
    })
    router.Use(cors.Handler)
    router.Use(middleware.RequestID)
    router.Use(middleware.Recoverer)
    router.Post("/orders", api.GetOrder(RequestValidator, &Cache, Storage))
    go http.ListenAndServe(":8000", router)

    // main loop
    ticker := time.NewTicker(Conf.TSUpdateInterval)
    var tstamp time.Time
    for {
        logger.Info("Running...")
        select {
        case intError = <-Errors:
            logger.Error(fmt.Sprintf("main | Error: %s", intError.Error()))
            var DBCritical *services.DBConnectionLost
            switch intError {
                case DBCritical: 
                    if _, err := Storage.TestConnection(); err != nil {
                        logger.Error("DB connection lost...")
                        return
                    }
                    logger.Info("DB connection found...")
                default:
                    logger.Error(fmt.Sprintf("[MAIN] Trapped: %v | %+v", intError, intError))
                    break
            }
        case tstamp = <-ticker.C:
            // sync cache with db
            // write new ts into .created file
            LogStateSync()
            services.UpdateTimestamp(tstamp, &logger)
        }
    }
}
