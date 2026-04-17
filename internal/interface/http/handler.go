package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

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
	r.Post("/shipments", h.create)
	r.Get("/shipments/{id}", h.getByID)
	r.Patch("/shipments/{id}/status", h.updateStatus)
	r.Post("/shipments/{id}/confirm", h.confirmDelivery)
}

func (h *ShipmentHandler) create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		OrderID        string         `json:"order_id"`
		OrderCreatedAt time.Time      `json:"order_created_at"`
		Address        domain.Address `json:"address"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	s, err := h.uc.CreateShipment(r.Context(), usecase.CreateShipmentInput{
		OrderID:        body.OrderID,
		OrderCreatedAt: body.OrderCreatedAt,
		Address:        body.Address,
	})
	if err != nil {
		respond(w, http.StatusInternalServerError, errBody(err))
		return
	}
	respond(w, http.StatusCreated, s)
}

func (h *ShipmentHandler) getByID(w http.ResponseWriter, r *http.Request) {
	s, err := h.uc.GetShipment(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			respond(w, http.StatusNotFound, errBody(err))
			return
		}
		respond(w, http.StatusInternalServerError, errBody(err))
		return
	}
	respond(w, http.StatusOK, s)
}

func (h *ShipmentHandler) updateStatus(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Status domain.DeliveryStatus `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	s, err := h.uc.UpdateShipmentStatus(r.Context(), usecase.UpdateShipmentStatusInput{
		ShipmentID: chi.URLParam(r, "id"),
		NewStatus:  body.Status,
	})
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			respond(w, http.StatusNotFound, errBody(err))
			return
		}
		if errors.Is(err, domain.ErrInvalidTransition) {
			respond(w, http.StatusUnprocessableEntity, errBody(err))
			return
		}
		respond(w, http.StatusInternalServerError, errBody(err))
		return
	}
	respond(w, http.StatusOK, s)
}

func (h *ShipmentHandler) confirmDelivery(w http.ResponseWriter, r *http.Request) {
	s, err := h.uc.ConfirmDelivery(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			respond(w, http.StatusNotFound, errBody(err))
			return
		}
		respond(w, http.StatusInternalServerError, errBody(err))
		return
	}
	respond(w, http.StatusOK, s)
}

func respond(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func errBody(err error) map[string]string {
	return map[string]string{"error": err.Error()}
}
