package main

// import "time"

type Account struct {
	ID     string  `json:"id"` // these are the json serialization mapping of ID --> id
	Name   string  `json:"name"`
	Orders []Order `json:"orders"`
}

// type Order struct {
// 	ID         string         `json:"id"`
// 	CreatedAt  time.Time      `json:"createdAt"`
// 	TotalPrice float64        `json:"totalPrice"`
// 	Products   []OrderProduct `json:"products"`
// }

// type OrderProduct struct {
// 	ID          string  `json:"id"`
// 	Name        string  `json:"name"`
// 	Description string  `json:"description"`
// 	Price       float64 `json:"price"`
// 	Quantity    int     `json:"quantity"`
// }
