package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"shipment_ms/internal/domain"
	"shipment_ms/internal/repository"
	"shipment_ms/internal/usecase"
)

type ShipmentHandler struct {
	uc *usecase.ShipmentUseCase
}

func NewShipmentHandler(uc *usecase.ShipmentUseCase) *ShipmentHandler {
	return &ShipmentHandler{uc: uc}
}

func (h *ShipmentHandler) RegisterRoutes(r chi.Router) {
	r.Post("/shipments", h.createShipment)
	r.Get("/shipments/{id}", h.getShipment)
	r.Patch("/shipments/status", h.updateShipmentStatus)
	r.Post("/shipments/confirm-delivery", h.confirmDelivery)
}

// POST /shipments
func (h *ShipmentHandler) createShipment(w http.ResponseWriter, r *http.Request) {
	var req createShipmentRequest
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := req.validate(); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	s, err := h.uc.CreateShipment(r.Context(), usecase.CreateShipmentInput{
		OrderID:        req.OrderID,
		OrderCreatedAt: req.OrderCreatedAt,
		Address:        req.Address.toDomain(),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusCreated, toShipmentResponse(s))
}

// GET /shipments/{id}
func (h *ShipmentHandler) getShipment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, errors.New("id is required"))
		return
	}

	s, err := h.uc.GetShipment(r.Context(), id)
	if err != nil {
		writeError(w, statusFor(err), err)
		return
	}

	writeJSON(w, http.StatusOK, toShipmentResponse(s))
}

// PATCH /shipments/status
func (h *ShipmentHandler) updateShipmentStatus(w http.ResponseWriter, r *http.Request) {
	var req updateShipmentStatusRequest
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := req.validate(); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	s, err := h.uc.UpdateShipmentStatus(r.Context(), usecase.UpdateShipmentStatusInput{
		ShipmentID: req.ShipmentID,
		NewStatus:  req.Status,
	})
	if err != nil {
		writeError(w, statusFor(err), err)
		return
	}

	writeJSON(w, http.StatusOK, toShipmentResponse(s))
}

// POST /shipments/confirm-delivery
func (h *ShipmentHandler) confirmDelivery(w http.ResponseWriter, r *http.Request) {
	var req confirmDeliveryRequest
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := req.validate(); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	s, err := h.uc.ConfirmDelivery(r.Context(), req.ShipmentID)
	if err != nil {
		// A Temporal signal failure is non-fatal: the shipment is already
		// confirmed in the DB. Surface the warning in the response body but
		// still return 200 with the persisted record.
		if s != nil {
			writeJSON(w, http.StatusOK, struct {
				Data    shipmentResponse `json:"data"`
				Warning string           `json:"warning"`
			}{
				Data:    toShipmentResponse(s),
				Warning: fmt.Sprintf("delivery confirmed but downstream signal failed: %s", err),
			})
			return
		}
		writeError(w, statusFor(err), err)
		return
	}

	writeJSON(w, http.StatusOK, toShipmentResponse(s))
}

// ── shared helpers ────────────────────────────────────────────────────────────

func decode(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("malformed JSON: %w", err)
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, errorResponse{Error: err.Error()})
}

// statusFor maps well-known domain/repository errors to HTTP status codes.
func statusFor(err error) int {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, domain.ErrInvalidTransition):
		return http.StatusUnprocessableEntity
	default:
		return http.StatusInternalServerError
	}
}
