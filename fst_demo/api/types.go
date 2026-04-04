package main

import db "example.com/fst_demo/db"

type CreateHouseRequest struct {
	Price       int         `json:"price"`
	Location    db.Location `json:"location"`
	Topology    string      `json:"topology"`
	Description string      `json:"description"`
}

type LoginRequest struct {
	Username string `json:"username"`
}

type CreateFilterRequest struct {
	UserID      string      `json:"user_id"`
	Location    db.Location `json:"location"`
	Topology    string      `json:"topology"`
	PriceBucket string      `json:"price_bucket,omitempty"`
	MinPrice    int         `json:"min_price"`
	MaxPrice    int         `json:"max_price"`
}
