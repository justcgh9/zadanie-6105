package tender

import (
	"encoding/json"
	serrors "errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"tender_system/internal/lib/errors"
	"tender_system/internal/models/tender"
	"tender_system/internal/models/user"
	"tender_system/internal/storage/postgres"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

type TenderSaver interface {
	SaveTender(ten tender.TenderRequest) (tender.TenderResponse, error)
	CheckOrganizationResponsible(username, organization_id string) (bool, error)
}

type TenderGetter interface {
	ReadTenders(limit, offset int, serviceType string) ([]tender.TenderResponse, error)
	FetchUserOrganization(username string) (string, error)
}

type MyTenderGetter interface {
	FetchUser(username string) (user.User, error)
	ReadMyTenders(creator string, limit int, offset int) ([]tender.TenderResponse, error)
	FetchUserOrganization(username string) (string, error)
}

type TenderStatusGetter interface {
	FetchUser(username string) (user.User, error)
	FetchUserOrganization(username string) (string, error)
	ReadTenderStatus(tenderId, username string) (string, error)
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
		serviceType := r.URL.Query().Get("service_type")
		if err := validateServiceType(serviceType); err != nil && serviceType != "" {
			log.Error("Incorrect service type")
			render.Status(r, 400)
			render.JSON(w, r, errors.NewHttpError("Incorrect service type"))
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

		resp, err := tenderGetter.ReadTenders(limit, offset, serviceType)
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

		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()

		err := decoder.Decode(&req)
		if err != nil {
			log.Error("Unknown Fields in request body")
			render.Status(r, 400)
			render.JSON(w, r, errors.NewHttpError(err.Error()))
			return
		}

		//TODO: Validate serviceType and other fields to be non-null

		err = validate.Struct(req)
		if err != nil {
			render.Status(r, 400)
			render.JSON(w, r, errors.NewHttpError("One of the fields is empty"))
			return
		}

		err = validateServiceType(req.ServiceType)
		if err != nil {
			render.Status(r, 400)
			render.JSON(w, r, errors.NewHttpError("Incorrect Service Type"))
			return
		}

		resp, err := tenderSaver.SaveTender(req)
		if err != nil {
			switch {
			case serrors.Is(err, postgres.ErrBadRequest):
				render.Status(r, 400)
			case serrors.Is(err, postgres.ErrUserNotFound):
				render.Status(r, 401)
			case serrors.Is(err, postgres.ErrForbidden):
				render.Status(r, 403)
			default:
				render.Status(r, 400)
			}
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

		resp, err := myTenderGetter.ReadMyTenders(username, limit, offset)
		if err != nil {
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

		tenderId := chi.URLParam(r, "tenderId")
		if tenderId == "" {
			render.Status(r, 400)
			render.JSON(w, r, errors.NewHttpError("The tender id is invalid"))
			return
		}

		status, err := tenderStatusGetter.ReadTenderStatus(tenderId, username)
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

		resp, err := tenderStatusPutter.UpdateTenderStatus(tenderId, status, username)
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
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()

		err := decoder.Decode(&patchRequest)
		if err != nil {
			render.Status(r, 400)
			render.JSON(w, r, errors.NewHttpError(err.Error()))
			return
		}
		if patchRequest.Name == "" && patchRequest.Description == "" && patchRequest.ServiceType == "" {
			render.Status(r, 400)
			render.JSON(w, r, errors.NewHttpError("The request body is empty"))
			return
		}

		if patchRequest.ServiceType != "" {
			err := validateServiceType(patchRequest.ServiceType)
			if err != nil {
				log.Error("Value error", slog.Attr{Key: "error", Value: slog.StringValue(err.Error())})
				render.Status(r, 400)
				render.JSON(w, r, errors.NewHttpError(err.Error()))
				return
			}
		}

		resp, err := tenderPatcher.PatchTender(tenderId, username, patchRequest.Name, patchRequest.Description, patchRequest.ServiceType)
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

		resp, err := tenderRollerBack.RollbackTender(tenderId, username, intVersion)
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
