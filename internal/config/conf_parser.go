package config

import (
    "fmt"
    "time"
    "os"
    "log"

    "github.com/ilyakaznacheev/cleanenv"
)

// app config
type AppConfig struct {
    Env string `yaml:"env" env-default:"error"`
    Encoding string `yaml:"encoding" env-default:"utf-8"`
    ApiVersion string `yaml:"api_version"`
    OnPanic string `yaml:"on_panic"`
    StoragePoolSize int `yaml:"storage_pool_size"`
    TSUpdateInterval time.Duration `yaml:"timestamp_interval"`
    RestoreRecordsLimit int `yaml:"restore_rec_limit"`
    HTTPConf HTTPConfig `yaml:"http_server"`
    DBConf DBEngineConf `yaml:"dbengine"`
    StanConf StanConfig `yaml:"stan_server"`
    CacheConf CacheConfig `yaml:"memcache"`
}

// http-server config
type HTTPConfig struct {
    Port string `yaml:"port"`
    Host string `yaml:"host"`
    ResponseTimeout time.Duration `yaml:"resp_timeout"`
    KeepAlive bool `yaml:"keep_alive"`
    AliveTime time.Duration `yaml:"alive_time"`
}

// db config
type DBEngineConf struct {
    Driver string `yaml:"driver"`
    Port string `yaml:"port"`
    Host string `yaml:"host"`
    DBName string `yaml:"dbname"`
    Passwd string `yaml:"passwd"`
    Db_admin string `yaml:"db_admin"`
    MaxPool int `yaml:"max_pool"`
    Timeout time.Duration `yaml:"timeout"`
    ConnRetry int `yaml:"conn_retry"`
}

// some parameters for connect to STAN
type StanConfig struct {
    Ask_wt time.Duration `yaml:"ask_wait"`
    ChannelName string `yaml:"channel_name"`
    DurableName string `yaml:"durable_name"`
    Cluster_id string `yaml:"cluster_id"`
    Client_id string `yaml:"client_id"`
}

type CacheConfig struct {
    Size int `yaml:"size"`
    Exp_time time.Duration `yaml:"expiration_time"`
}

// build config struct
func MustBuildConfig(envKey string) *AppConfig {
    conf_path := os.Getenv(envKey)
        if conf_path == "" {
            log.Fatal("Empty path parameter")
        }
    if _, err := os.Stat(conf_path); os.IsNotExist(err) {
        msg := fmt.Sprintf("Path: %s is not exists.", conf_path)
        log.Fatal(msg)
    }
    var cfg AppConfig
    if err := cleanenv.ReadConfig(conf_path, &cfg); err != nil {
        msg := fmt.Sprintf("Unreadable: %s", conf_path)
        log.Fatal(msg)
    }
    return &cfg
}
