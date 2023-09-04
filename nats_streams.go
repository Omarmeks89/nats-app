package main

import (
    "fmt"
    stan "github.com/nats-io/stan.go"
)

type Subscriber interface {
    Connect(a ...any) error
    Subscribe(a ...any) error
    Unsubscribe(a ...any) error
    Close() error
    Fetch() (any, error)
    // answer to server that we`re alive
    AnswAlive()
}
