package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"

	"shipment_ms/internal/domain"
	"shipment_ms/internal/infrastructure/temporal"
	"shipment_ms/internal/repository"
)

type ShipmentUseCase struct {
	repo     repository.ShipmentRepository
	temporal *temporal.Client
}

func NewShipmentUseCase(repo repository.ShipmentRepository, t *temporal.Client) *ShipmentUseCase {
	return &ShipmentUseCase{repo: repo, temporal: t}
}

func (uc *ShipmentUseCase) CreateShipment(ctx context.Context, orderID string, addr domain.Address) (*domain.Shipment, error) {
	s := &domain.Shipment{
		ID:        uuid.NewString(),
		OrderID:   orderID,
		Address:   addr,
		Status:    domain.Created,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	return uc.repo.Create(ctx, s)
}

func (uc *ShipmentUseCase) GetShipment(ctx context.Context, id string) (*domain.Shipment, error) {
	return uc.repo.GetByID(ctx, id)
}

func (uc *ShipmentUseCase) UpdateShipment(ctx context.Context, s *domain.Shipment) (*domain.Shipment, error) {
	return uc.repo.Update(ctx, s)
}
