package tender

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"tender_system/internal/lib/errors"
	"tender_system/internal/models/tender"
	"tender_system/internal/models/user"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type TenderSaver interface {
	SaveTender(ten tender.TenderRequest) (tender.TenderResponse, error)
}

type TenderGetter interface {
	ReadTenders(limit, offset int) ([]tender.TenderResponse, error)
}

type MyTenderGetter interface {
	FetchUser(username string) (user.User, error)
	ReadMyTenders(creator string, limit int, offset int) ([]tender.TenderResponse, error)
	FetchUserOrganization(username string) (string, error)
}

type TenderStatusGetter interface {
	FetchUser(username string) (user.User, error)
	FetchUserOrganization(username string) (string, error)
	ReadTenderStatus(tenderId string) (string, string, error)
}

type TenderStatusPutter interface {
	FetchUser(username string) (user.User, error)
	FetchUserOrganization(username string) (string, error)
	UpdateTenderStatus(tenderId, status, username string) (tender.TenderResponse, error)
}

type TenderPatcher interface {
	FetchUser(username string) (user.User, error)
	FetchUserOrganization(username string) (string, error)
	PatchTender(tenderId, username, name, description, serviceType string) (tender.TenderResponse, error)
}

type TendetRollerBack interface {
	FetchUser(username string) (user.User, error)
	FetchUserOrganization(username string) (string, error)
	RollbackTender(tenderId, username string, version int) (tender.TenderResponse, error)
}

func NewGetTenders(log *slog.Logger, tenderGetter TenderGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		resp, err := tenderGetter.ReadTenders(limit, offset)
		if err != nil {
			log.Error("Failed to read tenders", slog.Attr{Key: "error", Value: slog.StringValue(err.Error())})
			render.Status(r, 500)
			render.JSON(w, r, errors.NewHttpError(err.Error()))
			return
		}

		render.JSON(w, r, resp)

	}
}

func NewPostTender(log *slog.Logger, tenderSaver TenderSaver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req tender.TenderRequest

		err := render.DecodeJSON(r.Body, &req)
		if err != nil {
			log.Error("Failed to decode request body")
			render.Status(r, 400)
			render.JSON(w, r, errors.NewHttpError("Failed to decode request body"))
			return
		}

		//TODO: Validate serviceType and other fields to be non-null

		resp, err := tenderSaver.SaveTender(req)
		if err != nil {
			log.Error("Failed to save tender", slog.Attr{Key: "error", Value: slog.StringValue(err.Error())})
			render.Status(r, 500)
			render.JSON(w, r, errors.NewHttpError(err.Error()))
			return
		}

		render.JSON(w, r, resp)
	}
}

func NewGetMyTenders(log *slog.Logger, myTenderGetter MyTenderGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := r.URL.Query().Get("username")
		if username == "" {
			render.Status(r, 401)
			render.JSON(w, r, errors.NewHttpError("The Username is empty"))
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

		_, err = myTenderGetter.FetchUser(username)
		if err != nil {
			log.Error("Incorrect user information", slog.Attr{Key: "error", Value: slog.StringValue(err.Error())})
			render.Status(r, 401)
			render.JSON(w, r, errors.NewHttpError("Incorrect user information"))
			return
		}

		_, err = myTenderGetter.FetchUserOrganization(username)
		if err != nil {
			log.Error("User is not a responsible", slog.Attr{Key: "error", Value: slog.StringValue(err.Error())})
			render.Status(r, 401)
			render.JSON(w, r, errors.NewHttpError("User is not a responsible"))
			return
		}

		resp, err := myTenderGetter.ReadMyTenders(username, limit, offset)
		if err != nil {
			log.Error("Error reading tenders", slog.Attr{Key: "error", Value: slog.StringValue(err.Error())})
			render.Status(r, 401)
			render.JSON(w, r, errors.NewHttpError(err.Error()))
			return
		}

		render.JSON(w, r, resp)
	}
}

func NewGetTenderStatus(log *slog.Logger, tenderStatusGetter TenderStatusGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := r.URL.Query().Get("username")
		if username == "" {
			render.Status(r, 401)
			render.JSON(w, r, errors.NewHttpError("The Username is empty"))
			return
		}

		tenderId := chi.URLParam(r, "tenderId")
		if tenderId == "" {
			render.Status(r, 400)
			render.JSON(w, r, errors.NewHttpError("The tender id is invalid"))
			return
		}

		_, err := tenderStatusGetter.FetchUser(username)
		if err != nil {
			log.Error("Incorrect user information", slog.Attr{Key: "error", Value: slog.StringValue(err.Error())})
			render.Status(r, 401)
			render.JSON(w, r, errors.NewHttpError("Incorrect user information"))
			return
		}

		_, err = tenderStatusGetter.FetchUserOrganization(username)
		if err != nil {
			log.Error("User is not a responsible", slog.Attr{Key: "error", Value: slog.StringValue(err.Error())})
			render.Status(r, 401)
			render.JSON(w, r, errors.NewHttpError("User is not a responsible"))
			return
		}

		status, name, err := tenderStatusGetter.ReadTenderStatus(tenderId)
		if err != nil {
			log.Error("Tender does not exist", slog.Attr{Key: "error", Value: slog.StringValue(err.Error())})
			render.Status(r, 404)
			render.JSON(w, r, errors.NewHttpError("Tender does not exist"))
			return
		}

		if name != username {
			log.Error("given user is not creator of this tender")
			render.Status(r, 403)
			render.JSON(w, r, errors.NewHttpError("given user is not creator of this tender"))
			return
		}

		render.JSON(w, r, status)
	}
}

func NewPutTenderStatus(log *slog.Logger, tenderStatusPutter TenderStatusPutter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := r.URL.Query().Get("username")
		if username == "" {
			render.Status(r, 401)
			render.JSON(w, r, errors.NewHttpError("The Username is empty"))
			return
		}

		tenderId := chi.URLParam(r, "tenderId")
		if tenderId == "" {
			render.Status(r, 400)
			render.JSON(w, r, errors.NewHttpError("The tender id is invalid"))
			return
		}

		status := r.URL.Query().Get("status")
		err := validateStatus(status)
		if err != nil {
			render.Status(r, 400)
			render.JSON(w, r, errors.NewHttpError("Invalid status"))
			return
		}

		_, err = tenderStatusPutter.FetchUser(username)
		if err != nil {
			log.Error("Incorrect user information", slog.Attr{Key: "error", Value: slog.StringValue(err.Error())})
			render.Status(r, 401)
			render.JSON(w, r, errors.NewHttpError("Incorrect user information"))
			return
		}

		_, err = tenderStatusPutter.FetchUserOrganization(username)
		if err != nil {
			log.Error("User is not a responsible", slog.Attr{Key: "error", Value: slog.StringValue(err.Error())})
			render.Status(r, 401)
			render.JSON(w, r, errors.NewHttpError("User is not a responsible"))
			return
		}

		resp, err := tenderStatusPutter.UpdateTenderStatus(tenderId, status, username)
		if err != nil {

			//TODO: Add proper error handling
			log.Error("Error putting status", slog.Attr{Key: "error", Value: slog.StringValue(err.Error())})
			render.Status(r, 403)
			render.JSON(w, r, errors.NewHttpError("Error putting status"))
			return
		}

		render.JSON(w, r, resp)
	}
}

func NewPatchTender(log *slog.Logger, tenderPatcher TenderPatcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := r.URL.Query().Get("username")
		if username == "" {
			render.Status(r, 401)
			render.JSON(w, r, errors.NewHttpError("The Username is empty"))
			return
		}

		tenderId := chi.URLParam(r, "tenderId")
		if tenderId == "" {
			render.Status(r, 400)
			render.JSON(w, r, errors.NewHttpError("The tender id is invalid"))
			return
		}

		var patchRequest tender.TenderPatchRequest
		err := render.DecodeJSON(r.Body, &patchRequest)
		if err != nil {
			render.Status(r, 400)
			render.JSON(w, r, errors.NewHttpError(err.Error()))
		}
		if patchRequest.Name == "" && patchRequest.Description == "" && patchRequest.ServiceType == "" {
			render.Status(r, 400)
			render.JSON(w, r, errors.NewHttpError("The request body is empty"))
		}

		if patchRequest.ServiceType != "" {
			err := validateServiceType(patchRequest.ServiceType)
			if err != nil {
				log.Error("Value error", slog.Attr{Key: "error", Value: slog.StringValue(err.Error())})
				render.Status(r, 401)
				render.JSON(w, r, errors.NewHttpError(err.Error()))
				return
			}
		}

		_, err = tenderPatcher.FetchUser(username)
		if err != nil {
			log.Error("Incorrect user information", slog.Attr{Key: "error", Value: slog.StringValue(err.Error())})
			render.Status(r, 401)
			render.JSON(w, r, errors.NewHttpError("Incorrect user information"))
			return
		}

		_, err = tenderPatcher.FetchUserOrganization(username)
		if err != nil {
			log.Error("User is not a responsible", slog.Attr{Key: "error", Value: slog.StringValue(err.Error())})
			render.Status(r, 401)
			render.JSON(w, r, errors.NewHttpError("User is not a responsible"))
			return
		}

		resp, err := tenderPatcher.PatchTender(tenderId, username, patchRequest.Name, patchRequest.Description, patchRequest.ServiceType)
		if err != nil {
			log.Error("Error patching tender", slog.Attr{Key: "error", Value: slog.StringValue(err.Error())})
			render.Status(r, 401)
			render.JSON(w, r, errors.NewHttpError("Error patching tender"))
			return
		}

		render.JSON(w, r, resp)
	}
}

func NewRollbackTender(log *slog.Logger, tenderRollerBack TendetRollerBack) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := r.URL.Query().Get("username")
		if username == "" {
			render.Status(r, 401)
			render.JSON(w, r, errors.NewHttpError("The Username is empty"))
			return
		}

		tenderId := chi.URLParam(r, "tenderId")
		if tenderId == "" {
			render.Status(r, 400)
			render.JSON(w, r, errors.NewHttpError("The tender id is invalid"))
			return
		}

		version := chi.URLParam(r, "version")
		intVersion, err := strconv.Atoi(version)
		if err != nil {
			render.Status(r, 400)
			render.JSON(w, r, errors.NewHttpError("The version is invalid"))
			return
		}

		_, err = tenderRollerBack.FetchUser(username)
		if err != nil {
			log.Error("Incorrect user information", slog.Attr{Key: "error", Value: slog.StringValue(err.Error())})
			render.Status(r, 401)
			render.JSON(w, r, errors.NewHttpError("Incorrect user information"))
			return
		}

		_, err = tenderRollerBack.FetchUserOrganization(username)
		if err != nil {
			log.Error("User is not a responsible", slog.Attr{Key: "error", Value: slog.StringValue(err.Error())})
			render.Status(r, 401)
			render.JSON(w, r, errors.NewHttpError("User is not a responsible"))
			return
		}

		resp, err := tenderRollerBack.RollbackTender(tenderId, username, intVersion)
		if err != nil {
			log.Error("Error rolling back", slog.Attr{Key: "error", Value: slog.StringValue(err.Error())})
			render.Status(r, 401)
			render.JSON(w, r, errors.NewHttpError("Error rolling back"))
			return
		}

		render.JSON(w, r, resp)
	}
}

func validateStatus(status string) error {
	if status != "Created" && status != "Published" && status != "Closed" {
		return fmt.Errorf("invalid status parameter")
	}

	return nil
}

func validateServiceType(stype string) error {
	if stype != "Delivery" && stype != "Manufacture" && stype != "Construction" {
		return fmt.Errorf("invalid serviceType parameter")
	}
	return nil
}
