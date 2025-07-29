package main

// import "time"

type Account struct {
	ID     string  `json:"id"` // these are the json serialization mapping of ID --> id
	Name   string  `json:"name"`
	Orders []Order `json:"orders"`
}
