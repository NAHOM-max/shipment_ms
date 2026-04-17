package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"shipment_ms/internal/domain"
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
	r.Get("/shipments", h.list)
	r.Get("/shipments/{id}", h.getByID)
	r.Patch("/shipments/{id}/status", h.updateStatus)
}

func (h *ShipmentHandler) create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Origin      string `json:"origin"`
		Destination string `json:"destination"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	s, err := h.uc.CreateShipment(r.Context(), body.Origin, body.Destination)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respond(w, http.StatusCreated, s)
}

func (h *ShipmentHandler) list(w http.ResponseWriter, r *http.Request) {
	shipments, err := h.uc.ListShipments(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respond(w, http.StatusOK, shipments)
}

func (h *ShipmentHandler) getByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	s, err := h.uc.GetShipment(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respond(w, http.StatusOK, s)
}

func (h *ShipmentHandler) updateStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Status domain.Status `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if err := h.uc.UpdateStatus(r.Context(), id, body.Status); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func respond(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
