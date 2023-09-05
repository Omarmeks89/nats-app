package main

import (
    "os"
    "fmt"
    "nats_app/internal/config"
)

// mock main()
func main () {
    conf, err := config.MustBuildConfig()
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
}
