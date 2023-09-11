package main

import (
    "time"
)

type CustomerOrder struct {
    Order_id            string `json:"order_uid" validate:"required"`
    Track_numb          string `json:"track_number" validate:"required"`
    Entry               string `json:"entry" validate:"required"`
    Delivery            DeliveryModel `json:"delivery" validate:"required"`
    Payment             PaymentModel `json:"payment" validate:"required"`
    Items               []OrderItem `json:"items" validate:"required"`
    Locale              string `json:"locale"`
    IntSing             string `json:"internal_sinature" validate:"required"`
    CustomerId          string `json:"customer_id" validate:"required"`
    DeliveryServ        string `json:"delivery_service" validate:"required"`
    Shardkey            string `json:"shardkey"`
    SmId                int `json:"sm_id"`
    DateCreated         time.Time `json:"date_created" validate:"required"`
    OofShard            string `json:"oof_shard"`
}

type DeliveryModel struct {
    Name                string `json:"name"`
    Phone               string `json:"phone"`
    Zip                 string `json:"zip" validate:"required"`
    City                string `json:"city" validate:"required"`
    Address             string `json:"address" validate:"required"`
    Region              string `json:"region"`
    Email               string `json:"email" validate:"required"`
}

type PaymentModel struct {
    Trans               string `json:"transaction" validate:"required"`
    ReqId               string `json:"request_id" validate:"required"`
    Currency            string `json:"currency"`
    Provider            string `json:"provider"`
    Amount              int `json:"amount"`
    // Unix time
    PaymentDt           int64 `json:"payment_dt" validate:"required"`
    Bank                string `json:"bank" validate:"required"`
    DelivCost           int `json:"delivery_cost"`
    GoodsTotal          int `json:"goods_total"`
    CustomsFee          int `json:"customs_fee"`
}

type OrderItem struct {
    ChrtId              int `json:"chrt_id" validate:"required"`
    TrNumber            string `json:"track_number" validate:"required"`
    Price               int `json:"price"`
    Rid                 string `json:"rid"`
    Name                string `json:"name"`
    Sale                int `json:"sale"`
    Size                string `json:"size"`
    TotalPrice          int `json:"total_price"`
    NmId                int `json:"nm_id"`
    Brand               string `json:"brand"`
    Status              int `json:"status" validate:"required"`
}
