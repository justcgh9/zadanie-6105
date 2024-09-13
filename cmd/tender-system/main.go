package main

import (
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"tender_system/internal/http-server/handlers/api/bids"
	"tender_system/internal/http-server/handlers/api/ping"
	"tender_system/internal/http-server/handlers/api/tender"
	"tender_system/internal/storage/postgres"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
)

func main() {

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	err := godotenv.Load()
	if err != nil {
		log.Error("Failed to load .env", slog.Attr{Key: "error", Value: slog.StringValue(err.Error())})
	}

	connStr := os.Getenv("POSTGRES_CONN")
	log.Debug(connStr)
	storage, err := postgres.New(connStr)
	if err != nil {
		log.Error("Failed to connect to postgresql", slog.Attr{Key: "error", Value: slog.StringValue(err.Error())})
	}

	router := chi.NewRouter()

	router.Get("/api/tenders", tender.NewGetTenders(log, storage))
	router.Route("/api", func(r chi.Router) {
		// r.Post("/", )
		r.Get("/ping", ping.New(log))
		r.Route("/tenders", func(r chi.Router) {
			r.Post("/new", tender.NewPostTender(log, storage))
			r.Get("/my", tender.NewGetMyTenders(log, storage))
			r.Get("/{tenderId}/status", tender.NewGetTenderStatus(log, storage))
			r.Put("/{tenderId}/status", tender.NewPutTenderStatus(log, storage))
			r.Patch("/{tenderId}/edit", tender.NewPatchTender(log, storage))
			r.Put("/{tenderId}/rollback/{version}", tender.NewRollbackTender(log, storage))
		})
		r.Route("/bids", func(r chi.Router) {
			r.Post("/new", bids.NewPostBid(log, storage))
			r.Get("/my", bids.NewGetMyBids(log, storage))
			r.Get("/{tenderId}/list", bids.NewGetTenderBids(log, storage))
			r.Get("/{bidId}/status", bids.NewGetBidStatus(log, storage))
			r.Put("/{bidId}/status", bids.NewPutBidStatus(log, storage))
			r.Patch("/{bidId}/edit", bids.NewPatchBid(log, storage))
			r.Put("/{bidId}/feedback", bids.NewPutBidFeedback(log, storage))
			r.Put("/{bidId}/rollback/{version}", bids.NewRollbackBid(log, storage))
			r.Get("/{tenderId}/reviews", bids.NewReadBidFeedback(log, storage))
			r.Put("/{bidId}/submit_decision", bids.NewPutBidDecision(log, storage))
		})
	})

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Error("failed to start the server")
		}
	}()

	log.Info("starting server on port 8080")
	<-done
	log.Info("server stopped")
}
