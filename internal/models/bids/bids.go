package bids

import "time"

type BidRequest struct {
	Name        string `json:"name"`
	Description string `json:"desctiption"`
	TenderId    string `json:"tenderId"`
	AuthorType  string `json:"authorType"`
	AuthorId    string `json:"authorId"`
}

type Bid struct {
	Id          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"desctiption"`
	TenderId    string    `json:"tenderId"`
	AuthorType  string    `json:"authorType"`
	AuthorId    string    `json:"authorId"`
	Status      string    `json:"status"`
	Version     int       `json:"version"`
	CreatedAt   time.Time `json:"createdAt"`
}
