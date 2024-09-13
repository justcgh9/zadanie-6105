package ping

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/render"
)

func New(log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.api.ping.New"

		log := log.With(slog.String("op", op))
		log.Info("ping request")

		render.PlainText(w, r, "ok")
	}
}
