package services

import (
    "fmt"
    "context"

    "nats_app/internal/storage"
)

type Token uint8

// base worker interface
type Worker func(data interface{}, out chan<- interface{}, errs chan<- error)

type Order struct {
    oid string
    // payload -> json
    payload []byte
}

type SQLStoreService struct {
    ctx context.Context
    db storage.DBAdapter
    wPool chan Token
    outCh chan interface{}
    errCh chan<- error
}

// return channel for items that will 
// fetch data from db
func (srv *SQLStoreService) GetChannel() <-chan interface{} {
    return (*srv).outCh
}

func (srv *SQLStoreService) SaveOrder(o *Order) {
    go func(s *SQLStoreService, o *Order) {
        var t Token
        select {
        // try to fetch token from channel
        case t = <-(*s).wPool:
            query := fmt.Sprintf("SELECT * FROM orders; %+v", o)
            DBErr := (*s).db.Save(query)
            if DBErr != nil {
                select {
                case (*s).errCh<- DBErr:
                }
            }
            select {
            case <-(*s).ctx.Done():
                (*s).db.Close()
            case (*s).outCh<- o:
                // we can trust wPool
                (*s).wPool<- t
            }
        case <-(*s).ctx.Done():
            (*s).db.Close()
        }
        return
    }(srv, o)
    return
}

func (srv *SQLStoreService) FetchOrder(oid string) {
    go func(s *SQLStoreService, oid *string) {
        var t Token
        select {
        case t = <-(*s).wPool:
            // query sample
            query := fmt.Sprintf("SELECT payload FROM orders WHERE oid == %s", *oid)
            _, DBErr := (*s).db.FetchOne(query)
            if DBErr != nil {
                select {
                case (*s).errCh<- DBErr:
                }
            }
            select {
            case <-(*s).ctx.Done():
                (*s).db.Close()
            case (*s).outCh<- &Order{*oid, []byte{}}:
                (*s).wPool<- t
            }
        case <-(*s).ctx.Done():
            (*s).db.Close()
        }
        return
    }(srv, &oid)
    return
}
