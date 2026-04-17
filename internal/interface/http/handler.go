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
	r.Get("/shipments/{id}", h.getByID)
	r.Put("/shipments/{id}", h.update)
}

func (h *ShipmentHandler) create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		OrderID string       `json:"order_id"`
		Address domain.Address `json:"address"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	s, err := h.uc.CreateShipment(r.Context(), body.OrderID, body.Address)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respond(w, http.StatusCreated, s)
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

func (h *ShipmentHandler) update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var s domain.Shipment
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	s.ID = id
	updated, err := h.uc.UpdateShipment(r.Context(), &s)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respond(w, http.StatusOK, updated)
}

func respond(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
