package main

import (
    "strings"
    "math/rand"
    "time"
)

const (
    SymbsLowCase   string = "abcdefghijkmloprstuyxzqvw"
    SymbsUpperCase string = "ZAQXSWCDEVFRBGTNHYMJUKILOP"
    SymbsDigits    string = "1234567890"
)

func randStr(size int, symbsrc string) string {
    var result string
    var randSymbols []string
    for i := 0; i < size; i++ {
        s := symbsrc[rand.Intn(len(symbsrc) - 1)]
        randSymbols = append(randSymbols, string(s))
    }
    return strings.Join(randSymbols, result)
}

func LazySymbolsMix(a, b string) string {
    return a + b
}

func NewDeliveryModel() DeliveryModel {
    return DeliveryModel{
        Name:           "AnyClient",
        Phone:          "+" + randStr(10, SymbsDigits),
        Zip:            randStr(7, SymbsDigits),
        City:           "Moscow",
        Address:        "No address",
        Region:         "Moscow",
        Email:          randStr(9, LazySymbolsMix(SymbsDigits, SymbsLowCase)) + "@mail.ru",
    }
}

func NewPaymentModel() PaymentModel {
    return PaymentModel{
        Trans:          randStr(12, LazySymbolsMix(SymbsLowCase, SymbsDigits)) + "test",
        ReqId:          "",
        Currency:       "USD",
        Provider:       "wbpay",
        Amount:         rand.Intn(1235),
        PaymentDt:      time.Now().Unix(),
        Bank:           "horns and hooves",
        DelivCost:      rand.Intn(1000),
        GoodsTotal:     rand.Intn(400),
        CustomsFee:     0,
    }
}

func NewOrderItem() OrderItem {
    return OrderItem{
        ChrtId:         rand.Intn(999999),
        Price:          rand.Intn(999),
        Rid:            randStr(16, LazySymbolsMix(SymbsDigits, SymbsLowCase)),
        Name:           "Amasing stuff",
        Sale:           rand.Intn(1000),
        Size:           "0",
        TotalPrice:     rand.Intn(1999),
        NmId:           rand.Intn(2999999),
        Brand:          "Brand",
        Status:         rand.Intn(500),
    }
}

func NewOrder() CustomerOrder {
    rand.Seed(time.Now().UTC().UnixNano())
    var items []OrderItem
    delivery := NewDeliveryModel()
    payment := NewPaymentModel()

    entry := randStr(4, SymbsUpperCase)
    trnum := randStr(6, SymbsUpperCase)
    track_num := entry + trnum + "TEST"

    item := NewOrderItem()
    item.TrNumber = track_num
    items = append(items, item)
    return CustomerOrder{
        Order_id:       randStr(8, LazySymbolsMix(SymbsLowCase, SymbsDigits)),
        Track_numb:     track_num,
        Entry:          entry,
        Delivery:       delivery,
        Payment:        payment,
        Items:          items,
        Locale:         "ru",
        IntSing:        "",
        CustomerId:     randStr(8, LazySymbolsMix(SymbsUpperCase, SymbsDigits)),
        DeliveryServ:   "FIT",
        Shardkey:       randStr(1, SymbsDigits),
        SmId:           rand.Intn(100),
        DateCreated:    time.Now(),
        OofShard:       randStr(1, SymbsDigits),
    }
}
