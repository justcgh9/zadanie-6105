package bids

import "time"

type BidRequest struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description" validate:"required"`
	TenderId    string `json:"tenderId" validate:"required"`
	AuthorType  string `json:"authorType" validate:"required"`
	AuthorId    string `json:"authorId" validate:"required"`
}

type Bid struct {
	Id          string    `json:"id"`
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	Description string    `json:"description"`
	TenderId    string    `json:"tenderId"`
	AuthorType  string    `json:"authorType"`
	AuthorId    string    `json:"authorId"`
	Version     int       `json:"version"`
	CreatedAt   time.Time `json:"createdAt"`
}

type BidResponse struct {
	Id         string    `json:"id"`
	Name       string    `json:"name"`
	Status     string    `json:"status"`
	AuthorType string    `json:"authorType"`
	AuthorId   string    `json:"authorId"`
	Version    int       `json:"version"`
	CreatedAt  time.Time `json:"createdAt"`
}

type BidPatchRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type BidReviewResponse struct {
	Id          string    `json:"id"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
}
