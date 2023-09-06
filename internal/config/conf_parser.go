package config

import (
    "fmt"
    "time"
    "os"
    "errors"

    "github.com/ilyakaznacheev/cleanenv"
)

// app config
type AppConfig struct {
    Env string `yaml:"env" env-default:"error"`
    Encoding string `yaml:"encoding" env-default:"utf-8"`
    ApiVersion string `yaml:"api_version"`
    OnPanic string `yaml:"on_panic"`
    HTTPConf HTTPConfig `yaml:"http_server"`
    DBConf DBEngineConf `yaml:"dbengine"`
}

// http-server config
type HTTPConfig struct {
    Port string `yaml:"port"`
    Host string `yaml:"host"`
    ResponceTimeout time.Duration `yaml:"resp_timeout"`
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

// build config struct
func MustBuildConfig(envKey string) (*AppConfig, error) {
    conf_path := os.Getenv(envKey)
        if conf_path == "" {
            return &AppConfig{}, errors.New("path to config not set")
        }
    if _, err := os.Stat(conf_path); os.IsNotExist(err) {
        msg := fmt.Sprintf("Path: %s is not exists.", conf_path)
        return &AppConfig{}, errors.New(msg)
    }
    var cfg AppConfig
    if err := cleanenv.ReadConfig(conf_path, &cfg); err != nil {
        msg := fmt.Sprintf("Unreadable: %s", conf_path)
        return &cfg, errors.New(msg)
    }
    return &cfg, nil
}
