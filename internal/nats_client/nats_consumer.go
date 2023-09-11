package nats_client

import (
    "fmt"
    "os"
    "errors"
    "time"
    "log"
    "log/slog"
    "encoding/json"
    "context"

    stan "github.com/nats-io/stan.go"
    valid "github.com/go-playground/validator/v10"

    "nats_app/internal/storage"
    "nats_app/internal/services"
    "nats_app/internal/config"
)

const (
    FromTSMode string = "from_ts"
    FromLastReceived string = "from_last_recv"
    FromBegin string = "from_begin"
)

var (
    NatsConnFail = errors.New("Can`t connect to nats-streaming-server")
    InvChannelName = errors.New("Invalid channel name to subscribe")
    InvTimeStamp = errors.New("Invalid last message timestamp")
    NoSubscription = errors.New("No subscription created")
)

type Subscriber interface {
    SetStorageOnCallback(s services.AppStorage)
    SetLogger(l slog.Logger)
    RunFromLastId(id uint64)
    RunFromTimestamp(t time.Time)
    Run()
    Unsubscribe() error
    Disconnect() error
}

type AppConsumer struct {
    s stan.Conn
    sub stan.Subscription
    ask_wt time.Duration
    dur_name string
    channel string
    callback func(msg *stan.Msg)
    logger *slog.Logger
    errCh chan<- error
    ctx context.Context
}

func (nc AppConsumer) SetLogger(l *slog.Logger) {
    nc.logger = l
}

// called as go routine separately (inside stan)
func (nc *AppConsumer) SetStorageOnCallback(s *services.AppStorage) {
    cons := *nc
    val := valid.New()
    store := *s
    nc.callback = func(msg *stan.Msg) {
        var ordModel storage.CustomerOrder
        var errType error
        log := cons.logger
        if log == nil {
            // setup logger at place
            log = slog.New(
                slog.NewTextHandler(
                    os.Stdout,
                    &slog.HandlerOptions{Level: slog.LevelDebug},
                ),
            )
        }
        mark := "AppConsumer.Callback"
        select {
        case <-cons.ctx.Done():
            cons.sub.Close()
            return
        default:
            errType = json.Unmarshal((*msg).Data, &ordModel)
            if errType != nil {
                errType = fmt.Errorf(
                "%s: msg_id %d, error: %w",
                mark,
                (*msg).Sequence,
                errType,
            )
            } else {
                if errType = val.Struct(ordModel); errType != nil {
                    errType = fmt.Errorf(
                        "%s: msg_id %d, Validation error: %w",
                        mark,
                        (*msg).Sequence,
                        errType,
                    )
                }
            }
            if errType != nil {
                log.Debug(fmt.Sprintf("%s | Error: %+v", mark, errType))
                select {
                case cons.errCh<- errType:
                case <-cons.ctx.Done():
                    cons.sub.Close()
                    return
                }
            }
            msgForStorage := store.Convert(
                    (*msg).Sequence,
                    ordModel.Order_id,
                    &(*msg).Data,
                )
            store.SaveOrder(msgForStorage)
            report := fmt.Sprintf("%s | Order sent to DB. Client [%s], MsgNum [%d]...", mark, (*msg).Subject, (*msg).Sequence) 
            log.Debug(report)
            return
        }
    }
}

// will read messages from last received
func (nc AppConsumer) RunFromLastReseived() error {
    mark := "AppConsumer.RunFromLastReseived"
    var sub stan.Subscription
    var subErr error
    sub, subErr = nc.s.Subscribe(
        nc.channel,
        nc.callback,
        stan.DurableName(nc.dur_name),
        stan.StartWithLastReceived(),
        stan.AckWait(nc.ask_wt),
    )
    if subErr != nil {
        return fmt.Errorf("%s | Can`t subscribe %s. Error: %w", mark, nc.channel, subErr)
    }
    nc.sub = sub
    return nil
}

// will read messages from timestamp
func (nc AppConsumer) RunFromTimestamp(ts time.Time) error {
    mark := "AppConsumer.RunFromTimestamp"
    var sub stan.Subscription
    var subErr error
    sub, subErr = nc.s.Subscribe(
        nc.channel,
        nc.callback,
        stan.StartAtTime(ts),
        stan.DurableName(nc.dur_name),
        stan.AckWait(nc.ask_wt),
    )
    if subErr != nil {
        return fmt.Errorf("%s | Can`t subscribe %s. Error: %w", mark, nc.channel, subErr)
    }
    nc.sub = sub
    return nil
}

// will read all available messages
func (nc AppConsumer) Run() error {
    mark := "AppConsumer.Run"
    var sub stan.Subscription
    var subErr error
    sub, subErr = nc.s.Subscribe(
        nc.channel,
        nc.callback,
        stan.DeliverAllAvailable(),
        stan.DurableName(nc.dur_name),
        stan.AckWait(nc.ask_wt),
    )
    if subErr != nil {
        return fmt.Errorf("%s | Can`t subscribe %s. Error: %w", mark, nc.channel, subErr)
    }
    nc.sub = sub
    return nil
}

func (nc AppConsumer) Unsubscribe() error {
    mark := "AppConsumer.Unsubscribe"
    if nc.sub == nil {
        return NoSubscription
    }
    err := nc.sub.Unsubscribe()
    if err != nil {
        return fmt.Errorf("%s | Error: %w", mark, err)
    }
    nc.sub = nil
    return nil
}

func (nc AppConsumer) Disconnect() error {
    mark := "AppConsumer.Disconnect"
    if nc.sub == nil {
        return NoSubscription
    }
    err := nc.sub.Close()
    if err != nil {
        return fmt.Errorf("%s | Error: %w", mark, err)
    }
    nc.sub = nil
    return nil
}


func NewStanConsumer(
        ctx context.Context,
        errch chan<- error,
        s *config.StanConfig,
    ) AppConsumer {
    mark := "NewStanConsumer"
    if s.Cluster_id == "" || s.Client_id == "" {
        msg := fmt.Sprintf(
            "%s | Invalid creadentials: cluster_id = %s; client_id = %s",
            mark,
            s.Cluster_id,
            s.Client_id,
        )
        log.Fatal(msg)
    }
    conn, err := stan.Connect(s.Cluster_id, s.Client_id)
    if err != nil {
        err = fmt.Errorf("%s | Can`t connect to server. Error: %w", mark, err)
        log.Fatal(err)
    }
    return AppConsumer{
        s:               conn,
        sub:                nil,
        ask_wt:             s.Ask_wt,
        ctx:                ctx,
        channel:            s.ChannelName,
        dur_name:           s.DurableName,
        errCh:              errch,
    }
}
