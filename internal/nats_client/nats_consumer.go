package nats_client

import (
    "fmt"
    "errors"
    "time"

    stan "github.com/nats-io/stan.go"
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
    SubscribeDurable(ch, dur_name string, cb func(msg *stan.Msg)) error
    Unsubscribe() error
    Close() error
}

type NatsConsumer struct {
    s stan.Conn
    sub stan.Subscription
    // st_mode -> from which msg will we start and how
    // from timestamp or from last position
    st_mode string
    last_recv_ts time.Time
    last_recv_idx uint64
    ask_wt time.Duration
}

func (nc *NatsConsumer) SubscribeDurable(ch, dur_name string, cb func(msg *stan.Msg)) error {
    if ch == "" {
        return InvChannelName
    }
    var sub stan.Subscription
    var subErr error
    switch (*nc).st_mode {
    case FromTSMode:
        sub, subErr = (*nc).s.Subscribe(ch, cb, stan.StartAtTime((*nc).last_recv_ts), stan.DurableName(dur_name))
        if subErr != nil {
            return fmt.Errorf("Can`t subscribe on channel %s. Error: %w", ch, subErr)
        }
    case FromBegin:
        sub, subErr = (*nc).s.Subscribe(ch, cb, stan.DeliverAllAvailable(), stan.DurableName(dur_name))
        if subErr != nil {
            return fmt.Errorf("Can`t subscribe on channel %s. Error: %w", ch, subErr)
        }
    case FromLastReceived:
        sub, subErr = (*nc).s.Subscribe(ch, cb, stan.StartAtSequence((*nc).last_recv_idx), stan.DurableName(dur_name))
        if subErr != nil {
            return fmt.Errorf("Can`t subscribe on channel %s. Error: %w", ch, subErr)
        }
    }
    (*nc).sub = sub
    return nil
}

func (nc *NatsConsumer) Unsubscribe() error {
    if (*nc).sub == nil {
        return NoSubscription
    }
    err := (*nc).Unsubscribe()
    if err != nil {
        return fmt.Errorf("Can`t unsubscribe. Error: %w", err)
    }
    (*nc).sub = nil
    return nil
}

func (nc *NatsConsumer) Close() error {
    if (*nc).sub == nil {
        return NoSubscription
    }
    err := (*nc).Close()
    if err != nil {
        return fmt.Errorf("Can`t close subscription. Error: %w", err)
    }
    (*nc).sub = nil
    return nil
}


func NewNatsConsumer(s any) (Subscriber, error) {
    // s -> means settings for NATS
    if s.cluster_id == "" || s.client_id == "" {
        msg := fmt.Sprintf("Invalid creadentials:\n\tCluster: %s\n\tClient: %s", s.cluster_id, s.client_id)
        return nil, errors.New(msg)
    }
    conn, err := stan.Connect(s.cluster_id, s.client_id)
    if err != nil {
        return nil, fmt.Errorf("Can`t connect to server. Error: %w", err)
    }
    return NatsConsumer{
        conn:               conn,
        sub:                nil,
        st_mode:            s.St_mode,
        last_recv_ts:       s.Last_recv_ts,
        last_recv_idx:      uint64(s.Last_recv_idx),
        ask_wt:             s.Ask_wt,
    }, nil
}
