package model

import "time"

type Order struct {
	OrderUID          string    `json:"order_uid" validate:"required,uuid"`
	TrackNumber       string    `json:"track_number" validate:"required"`
	Entry             string    `json:"entry" validate:"required"`
	Locale            string    `json:"locale" validate:"required,min=2,max=5"`
	InternalSignature string    `json:"internal_signature" validate:"required"`
	CustomerID        string    `json:"customer_id" validate:"required"`
	DeliveryService   string    `json:"delivery_service" validate:"required"`
	Shardkey          string    `json:"shardkey" validate:"required"`
	SMID              int       `json:"sm_id" validate:"required,gte=0"`
	DateCreated       time.Time `json:"date_created" validate:"required"`
	OofShard          string    `json:"oof_shard" validate:"required"`
	Delivery          Delivery  `json:"delivery" validate:"required,dive"`
	Payment           Payment   `json:"payment" validate:"required,dive"`
	Items             []Item    `json:"items" validate:"required,dive,min=1"`
}

type Delivery struct {
	Name    string `json:"name" validate:"required,min=3,max=100"`
	Phone   string `json:"phone" validate:"required,numeric,len=11"`
	Zip     string `json:"zip" validate:"required,alphanum,max=10"`
	City    string `json:"city" validate:"required,min=3,max=100"`
	Address string `json:"address" validate:"required,min=5,max=255"`
	Region  string `json:"region" validate:"required,min=3,max=100"`
	Email   string `json:"email" validate:"omitempty,email"`
}

type Payment struct {
	Transaction  string  `json:"transaction" validate:"required"`
	RequestID    string  `json:"request_id" validate:"required"`
	Currency     string  `json:"currency" validate:"required,in=USD,RUB,EUR"`
	Provider     string  `json:"provider" validate:"required"`
	Amount       float64 `json:"amount" validate:"required,gte=0"`
	PaymentDT    int64   `json:"payment_dt" validate:"required,gte=0"`
	Bank         string  `json:"bank" validate:"required"`
	DeliveryCost float64 `json:"delivery_cost" validate:"required,gte=0"`
	GoodsTotal   int     `json:"goods_total" validate:"required,gte=0"`
	CustomFee    float64 `json:"custom_fee" validate:"required,gte=0"`
}

type Item struct {
	ChrtID      int     `json:"chrt_id" validate:"required,gte=0"`
	TrackNumber string  `json:"track_number" validate:"required"`
	Price       float64 `json:"price" validate:"required,gte=0"`
	RID         string  `json:"rid" validate:"required"`
	Name        string  `json:"name" validate:"required,min=1,max=255"`
	Sale        float64 `json:"sale" validate:"omitempty,gte=0,lte=100"`
	Size        string  `json:"size" validate:"required,min=1,max=10"`
	TotalPrice  float64 `json:"total_price" validate:"required,gte=0"`
	NMID        int     `json:"nm_id" validate:"required,gte=0"`
	Brand       string  `json:"brand" validate:"required,min=1,max=100"`
	Status      int     `json:"status" validate:"required,gte=0,lte=100"`
}
