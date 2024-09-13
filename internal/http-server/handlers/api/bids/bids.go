package bids

import (
	"encoding/json"
	serrors "errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"tender_system/internal/lib/errors"
	"tender_system/internal/models/bids"
	"tender_system/internal/storage/postgres"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type BidSaver interface {
	SaveBid(bid bids.BidRequest) (bids.BidResponse, error)
}

type MyBidsReader interface {
	ReadMyBids(username string, limit int, offset int) ([]bids.BidResponse, error)
}

type TenderBidsReader interface {
	ReadTenderBids(username string, tenderId string, limit int, offset int) ([]bids.BidResponse, error)
}

type BidStatusReader interface {
	GetBidStatus(bidId, username string) (string, error)
}

type BidStatusUpdater interface {
	ChangeBidStatus(bidId, status, username string) (bids.BidResponse, error)
}

type BidEditor interface {
	EditBid(bidId, username, name, desc string) (bids.BidResponse, error)
}

type BidFeedbackWriter interface {
	LeaveFeedback(bidId, bidFeedback, username string) (bids.BidResponse, error)
}

type BidRollerBack interface {
	RollbackBid(bidId, username string, version int) (bids.BidResponse, error)
}

type BidFeedbackReader interface {
	GetTenderReviews(tenderId, authorUsername, requesterUsername string, limit, offset int) ([]bids.BidReviewResponse, error)
}

type BidDecisionHandler interface {
	SubmitDecision(bidId, decision, username string) (bids.BidResponse, error)
}

func NewPostBid(log *slog.Logger, bidSaver BidSaver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req bids.BidRequest

		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()

		err := decoder.Decode(&req)
		if err != nil {
			log.Error("Error decoding request body")
			render.Status(r, 400)
			render.JSON(w, r, errors.NewHttpError("Error decoding request body"))
			return
		}

		err = validateBidRequest(req)
		if err != nil {
			log.Error(err.Error())
			render.Status(r, 400)
			render.JSON(w, r, errors.NewHttpError(err.Error()))
			return
		}

		resp, err := bidSaver.SaveBid(req)
		if err != nil {
			switch {
			case serrors.Is(err, postgres.ErrBadRequest):
				render.Status(r, 400)
			case serrors.Is(err, postgres.ErrUserNotFound):
				render.Status(r, 401)
			case serrors.Is(err, postgres.ErrForbidden):
				render.Status(r, 403)
			case serrors.Is(err, postgres.ErrNotFound):
				render.Status(r, 404)
			default:
				render.Status(r, 400)
			}
			render.JSON(w, r, errors.NewHttpError(err.Error()))
			return
		}

		render.JSON(w, r, resp)

	}
}

func NewGetMyBids(log *slog.Logger, myBidsReader MyBidsReader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var limit, offset int
		username := r.URL.Query().Get("username")
		if username == "" {
			render.JSON(w, r, make([]int, 0))
			return
		}
		var err error
		if r.URL.Query().Get("limit") == "" {
			limit = 5
		} else {
			limit, err = strconv.Atoi(r.URL.Query().Get("limit"))
			if err != nil {
				log.Error("Incorrect limit value")
				render.Status(r, 400)
				render.JSON(w, r, errors.NewHttpError("Incorrect limit value"))
				return
			}
		}
		if r.URL.Query().Get("offset") == "" {
			offset = 0
		} else {
			offset, err = strconv.Atoi(r.URL.Query().Get("offset"))
			if err != nil {
				log.Error("Incorrect offset value")
				render.Status(r, 400)
				render.JSON(w, r, errors.NewHttpError("Incorrect offset value"))
				return
			}
		}

		resp, err := myBidsReader.ReadMyBids(username, limit, offset)
		if err != nil {
			switch {
			case serrors.Is(err, postgres.ErrBadRequest):
				render.Status(r, 400)
			case serrors.Is(err, postgres.ErrUserNotFound):
				render.Status(r, 401)
			case serrors.Is(err, postgres.ErrForbidden):
				render.Status(r, 403)
			case serrors.Is(err, postgres.ErrNotFound):
				render.Status(r, 404)
			default:
				render.Status(r, 400)
			}
			render.JSON(w, r, errors.NewHttpError(err.Error()))
			return
		}

		render.JSON(w, r, resp)
	}
}

func NewGetTenderBids(log *slog.Logger, tenderBidsReader TenderBidsReader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var limit, offset int
		username := r.URL.Query().Get("username")
		if username == "" {
			render.Status(r, 401)
			render.JSON(w, r, errors.NewHttpError("The Username is empty"))
			return
		}
		var err error
		if r.URL.Query().Get("limit") == "" {
			limit = 5
		} else {
			limit, err = strconv.Atoi(r.URL.Query().Get("limit"))
			if err != nil {
				log.Error("Incorrect limit value")
				render.Status(r, 400)
				render.JSON(w, r, errors.NewHttpError("Incorrect limit value"))
				return
			}
		}
		if r.URL.Query().Get("offset") == "" {
			offset = 0
		} else {
			offset, err = strconv.Atoi(r.URL.Query().Get("offset"))
			if err != nil {
				log.Error("Incorrect offset value")
				render.Status(r, 400)
				render.JSON(w, r, errors.NewHttpError("Incorrect offset value"))
				return
			}
		}

		tenderId := chi.URLParam(r, "tenderId")
		if tenderId == "" {
			render.Status(r, 400)
			render.JSON(w, r, errors.NewHttpError("The tender id is invalid"))
			return
		}

		resp, err := tenderBidsReader.ReadTenderBids(username, tenderId, limit, offset)
		if err != nil {
			switch {
			case serrors.Is(err, postgres.ErrBadRequest):
				render.Status(r, 400)
			case serrors.Is(err, postgres.ErrUserNotFound):
				render.Status(r, 401)
			case serrors.Is(err, postgres.ErrForbidden):
				render.Status(r, 403)
			case serrors.Is(err, postgres.ErrNotFound):
				render.Status(r, 404)
			default:
				render.Status(r, 400)
			}
			render.JSON(w, r, errors.NewHttpError(err.Error()))
			return
		}

		render.JSON(w, r, resp)
	}
}

func NewGetBidStatus(log *slog.Logger, bidStatusReader BidStatusReader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bidId := chi.URLParam(r, "bidId")
		if bidId == "" {
			render.Status(r, 400)
			render.JSON(w, r, errors.NewHttpError("The bid id is invalid"))
			return
		}
		username := r.URL.Query().Get("username")
		if username == "" {
			render.Status(r, 401)
			render.JSON(w, r, errors.NewHttpError("The Username is empty"))
			return
		}
		resp, err := bidStatusReader.GetBidStatus(bidId, username)
		if err != nil {
			switch {
			case serrors.Is(err, postgres.ErrBadRequest):
				render.Status(r, 400)
			case serrors.Is(err, postgres.ErrUserNotFound):
				render.Status(r, 401)
			case serrors.Is(err, postgres.ErrForbidden):
				render.Status(r, 403)
			case serrors.Is(err, postgres.ErrNotFound):
				render.Status(r, 404)
			default:
				render.Status(r, 400)
			}
			render.JSON(w, r, errors.NewHttpError(err.Error()))
			return
		}
		render.JSON(w, r, resp)
	}
}

func NewPutBidStatus(log *slog.Logger, bidStatusUpdater BidStatusUpdater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bidId := chi.URLParam(r, "bidId")
		if bidId == "" {
			render.Status(r, 400)
			render.JSON(w, r, errors.NewHttpError("The bid id is invalid"))
			return
		}
		username := r.URL.Query().Get("username")
		if username == "" {
			render.Status(r, 401)
			render.JSON(w, r, errors.NewHttpError("The Username is empty"))
			return
		}
		status := r.URL.Query().Get("status")
		if status == "" || (status != "Created" && status != "Published" && status != "Canceled") {
			render.Status(r, 401)
			render.JSON(w, r, errors.NewHttpError("The status is wrong"))
			return
		}

		resp, err := bidStatusUpdater.ChangeBidStatus(bidId, status, username)
		if err != nil {
			switch {
			case serrors.Is(err, postgres.ErrBadRequest):
				render.Status(r, 400)
			case serrors.Is(err, postgres.ErrUserNotFound):
				render.Status(r, 401)
			case serrors.Is(err, postgres.ErrForbidden):
				render.Status(r, 403)
			case serrors.Is(err, postgres.ErrNotFound):
				render.Status(r, 404)
			default:
				render.Status(r, 400)
			}
			render.JSON(w, r, errors.NewHttpError(err.Error()))
			return
		}
		render.JSON(w, r, resp)
	}
}

func NewPatchBid(log *slog.Logger, bidEditor BidEditor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bidId := chi.URLParam(r, "bidId")
		if bidId == "" {
			render.Status(r, 400)
			render.JSON(w, r, errors.NewHttpError("The bid id is invalid"))
			return
		}
		username := r.URL.Query().Get("username")
		if username == "" {
			render.Status(r, 401)
			render.JSON(w, r, errors.NewHttpError("The Username is empty"))
			return
		}
		var req bids.BidPatchRequest
		err := render.DecodeJSON(r.Body, &req)
		if err != nil {
			log.Error(err.Error())
			render.Status(r, 400)
			render.JSON(w, r, errors.NewHttpError(err.Error()))
			return
		}

		if req.Name == "" && req.Description == "" {
			render.Status(r, 400)
			render.JSON(w, r, errors.NewHttpError("The request body is empty"))
			return
		}

		resp, err := bidEditor.EditBid(bidId, username, req.Name, req.Description)
		if err != nil {
			switch {
			case serrors.Is(err, postgres.ErrBadRequest):
				render.Status(r, 400)
			case serrors.Is(err, postgres.ErrUserNotFound):
				render.Status(r, 401)
			case serrors.Is(err, postgres.ErrForbidden):
				render.Status(r, 403)
			case serrors.Is(err, postgres.ErrNotFound):
				render.Status(r, 404)
			default:
				render.Status(r, 400)
			}
			render.JSON(w, r, errors.NewHttpError(err.Error()))
			return
		}
		render.JSON(w, r, resp)
	}
}

func NewPutBidDecision(log *slog.Logger, bidDecisionHandler BidDecisionHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bidId := chi.URLParam(r, "bidId")
		if bidId == "" {
			render.Status(r, 400)
			render.JSON(w, r, errors.NewHttpError("The bid id is invalid"))
			return
		}
		username := r.URL.Query().Get("username")
		if username == "" {
			render.Status(r, 401)
			render.JSON(w, r, errors.NewHttpError("The Username is empty"))
			return
		}
		decision := r.URL.Query().Get("decision")
		if decision == "" || (decision != "Approved" && decision != "Rejected") {
			render.Status(r, 400)
			render.JSON(w, r, errors.NewHttpError("The decision is wrong"))
			return
		}

		resp, err := bidDecisionHandler.SubmitDecision(bidId, decision, username)
		if err != nil {
			switch {
			case serrors.Is(err, postgres.ErrBadRequest):
				render.Status(r, 400)
			case serrors.Is(err, postgres.ErrUserNotFound):
				render.Status(r, 401)
			case serrors.Is(err, postgres.ErrForbidden):
				render.Status(r, 403)
			case serrors.Is(err, postgres.ErrNotFound):
				render.Status(r, 404)
			default:
				render.Status(r, 400)
			}
			render.JSON(w, r, errors.NewHttpError(err.Error()))
			return
		}
		render.JSON(w, r, resp)
	}
}

func NewPutBidFeedback(log *slog.Logger, bidFeedbackWriter BidFeedbackWriter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bidId := chi.URLParam(r, "bidId")
		if bidId == "" {
			render.Status(r, 400)
			render.JSON(w, r, errors.NewHttpError("The bid id is invalid"))
			return
		}
		username := r.URL.Query().Get("username")
		if username == "" {
			render.Status(r, 401)
			render.JSON(w, r, errors.NewHttpError("The Username is empty"))
			return
		}
		bidFeedback := r.URL.Query().Get("bidFeedback")
		if bidFeedback == "" {
			render.Status(r, 400)
			render.JSON(w, r, errors.NewHttpError("The bidFeedback is empty"))
			return
		}

		resp, err := bidFeedbackWriter.LeaveFeedback(bidId, bidFeedback, username)
		if err != nil {
			switch {
			case serrors.Is(err, postgres.ErrBadRequest):
				render.Status(r, 400)
			case serrors.Is(err, postgres.ErrUserNotFound):
				render.Status(r, 401)
			case serrors.Is(err, postgres.ErrForbidden):
				render.Status(r, 403)
			case serrors.Is(err, postgres.ErrNotFound):
				render.Status(r, 404)
			default:
				render.Status(r, 400)
			}
			render.JSON(w, r, errors.NewHttpError(err.Error()))
			return
		}
		render.JSON(w, r, resp)
	}
}

func NewRollbackBid(log *slog.Logger, bidRollerBack BidRollerBack) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bidId := chi.URLParam(r, "bidId")
		if bidId == "" {
			render.Status(r, 400)
			render.JSON(w, r, errors.NewHttpError("The bid id is invalid"))
			return
		}
		username := r.URL.Query().Get("username")
		if username == "" {
			render.Status(r, 401)
			render.JSON(w, r, errors.NewHttpError("The Username is empty"))
			return
		}
		version := chi.URLParam(r, "version")
		intVersion, err := strconv.Atoi(version)
		if err != nil {
			render.Status(r, 400)
			render.JSON(w, r, errors.NewHttpError("The version is invalid"))
			return
		}

		resp, err := bidRollerBack.RollbackBid(bidId, username, intVersion)
		if err != nil {
			switch {
			case serrors.Is(err, postgres.ErrBadRequest):
				render.Status(r, 400)
			case serrors.Is(err, postgres.ErrUserNotFound):
				render.Status(r, 401)
			case serrors.Is(err, postgres.ErrForbidden):
				render.Status(r, 403)
			case serrors.Is(err, postgres.ErrNotFound):
				render.Status(r, 404)
			default:
				render.Status(r, 400)
			}
			render.JSON(w, r, errors.NewHttpError(err.Error()))
			return
		}
		render.JSON(w, r, resp)
	}
}

func NewReadBidFeedback(log *slog.Logger, bidFeedbackReader BidFeedbackReader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenderId := chi.URLParam(r, "tenderId")
		if tenderId == "" {
			render.Status(r, 400)
			render.JSON(w, r, errors.NewHttpError("The bid id is invalid"))
			return
		}

		authorUsername := r.URL.Query().Get("authorUsername")
		if authorUsername == "" {
			render.Status(r, 401)
			render.JSON(w, r, errors.NewHttpError("The authorUsername is empty"))
			return
		}
		requesterUsername := r.URL.Query().Get("requesterUsername")
		if requesterUsername == "" {
			render.Status(r, 401)
			render.JSON(w, r, errors.NewHttpError("The requesterUsername is empty"))
			return
		}
		var limit, offset int
		var err error
		if r.URL.Query().Get("limit") == "" {
			limit = 5
		} else {
			limit, err = strconv.Atoi(r.URL.Query().Get("limit"))
			if err != nil {
				log.Error("Incorrect limit value")
				render.Status(r, 400)
				render.JSON(w, r, errors.NewHttpError("Incorrect limit value"))
				return
			}
		}
		if r.URL.Query().Get("offset") == "" {
			offset = 0
		} else {
			offset, err = strconv.Atoi(r.URL.Query().Get("offset"))
			if err != nil {
				log.Error("Incorrect offset value")
				render.Status(r, 400)
				render.JSON(w, r, errors.NewHttpError("Incorrect offset value"))
				return
			}
		}

		resp, err := bidFeedbackReader.GetTenderReviews(tenderId, authorUsername, requesterUsername, limit, offset)
		if err != nil {
			switch {
			case serrors.Is(err, postgres.ErrBadRequest):
				render.Status(r, 400)
			case serrors.Is(err, postgres.ErrUserNotFound):
				render.Status(r, 401)
			case serrors.Is(err, postgres.ErrForbidden):
				render.Status(r, 403)
			case serrors.Is(err, postgres.ErrNotFound):
				render.Status(r, 404)
			default:
				render.Status(r, 400)
			}
			render.JSON(w, r, errors.NewHttpError(err.Error()))
			return
		}
		render.JSON(w, r, resp)
	}
}

func validateBidRequest(bid bids.BidRequest) error {
	if bid.Name == "" || bid.Description == "" || bid.TenderId == "" || (bid.AuthorType != "Organization" && bid.AuthorType != "User") || bid.AuthorId == "" {
		fmt.Println(bid)
		return fmt.Errorf("invalid bid request body")
	}
	return nil
}
