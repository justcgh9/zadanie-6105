package bids

import (
	"log/slog"
	"net/http"
)

func NewPostBid(log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}

func NewGetMyBids(log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}

func NewGetTenderBids(log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}

func NewGetBidStatus(log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}

func NewPutBidStatus(log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}

func NewPatchBid(log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}

func NewPutBidDecision(log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}

func NewPutBidFeedback(log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}

func NewRollbackBid(log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}
