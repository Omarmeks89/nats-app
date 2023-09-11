package main

import (
    "os"
    "fmt"
    "time"
    "log/slog"
    "encoding/json"
    "math/rand"

    stan "github.com/nats-io/stan.go"
)

const (
    ClusterId string = "local"
    ClientId string = "Test_consumer"
    ChanName string = "orders"
    MaxAckInFlight int = 32
)

func main() {
    fmt.Println("Publisher configuring...")
    mark := "Stan-publisher.Main"
    logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
    Handler := func(Nuid string, err error) {
        mark := "Stan-publisher.Handler"
        fmt.Printf("%s | nuid = %s\n", mark, Nuid)
    }
    logger.Debug("Logger configured...")
    stan_conn, err := stan.Connect(ClusterId, ClientId, stan.MaxPubAcksInflight(MaxAckInFlight))
    if err != nil {
        msg := fmt.Sprintf("%s | Error %s", mark, err.Error())
        logger.Error(msg)
    }
    logger.Info("Connected...")
    logger.Info("Run publishing...")
    for {
        rand.Seed(127521)
        order := NewOrder()
        logger.Info(order.Order_id)
        JSONOrder, JSONErr := json.Marshal(order)
        if JSONErr != nil {
            msg := fmt.Sprintf("%s | Error %s", mark, JSONErr.Error())
            logger.Error(msg)
        }
        nuid, err := stan_conn.PublishAsync(ChanName, JSONOrder, Handler)
        if err != nil {
            msg := fmt.Sprintf("%s | Error %s", mark, err.Error())
            logger.Error(msg)
        }
        ShowNuid := fmt.Sprintf("%s | Message was sent. NUID %s", mark, nuid)
        logger.Debug(ShowNuid)
        time.Sleep(1 * time.Second)
    }
}
