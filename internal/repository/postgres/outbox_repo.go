package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"shipment_ms/internal/domain"
	"shipment_ms/internal/repository"
)

type outboxRepo struct {
	q *Queries
}

func NewOutboxRepository(pool *pgxpool.Pool) repository.OutboxRepository {
	return &outboxRepo{q: New(pool)}
}

func (r *outboxRepo) CreateOutboxEvent(ctx context.Context, e *domain.OutboxEvent) error {
	uid, err := parseUUID(e.ID)
	if err != nil {
		return err
	}
	payload, err := marshalPayload(e.Payload)
	if err != nil {
		return err
	}
	return r.q.InsertOutboxEvent(ctx, InsertOutboxEventParams{
		ID:            uid,
		AggregateType: e.AggregateType,
		AggregateID:   e.AggregateID,
		EventType:     e.EventType,
		Payload:       payload,
	})
}

func (r *outboxRepo) FetchPending(ctx context.Context, maxRetries, limit int) ([]*domain.OutboxEvent, error) {
	rows, err := r.q.FetchPendingOutboxEvents(ctx, FetchPendingOutboxEventsParams{
		RetryCount: int32(maxRetries),
		Limit:      int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("fetch pending outbox events: %w", err)
	}
	out := make([]*domain.OutboxEvent, len(rows))
	for i, row := range rows {
		out[i] = outboxToDomain(row)
	}
	return out, nil
}

func (r *outboxRepo) MarkSent(ctx context.Context, eventID string) error {
	uid, err := parseUUID(eventID)
	if err != nil {
		return err
	}
	return r.q.MarkOutboxEventSent(ctx, uid)
}

func (r *outboxRepo) MarkFailed(ctx context.Context, eventID string) error {
	uid, err := parseUUID(eventID)
	if err != nil {
		return err
	}
	return r.q.MarkOutboxEventFailed(ctx, uid)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func outboxToDomain(row OutboxEvent) *domain.OutboxEvent {
	return &domain.OutboxEvent{
		ID:            row.ID.String(),
		AggregateType: row.AggregateType,
		AggregateID:   row.AggregateID,
		EventType:     row.EventType,
		Payload:       row.Payload,
		Status:        domain.OutboxStatus(row.Status),
		RetryCount:    int(row.RetryCount),
		CreatedAt:     row.CreatedAt.Time,
		UpdatedAt:     row.UpdatedAt.Time,
	}
}

// marshalPayload accepts []byte (already JSON) or any value and returns JSON bytes.
func marshalPayload(v any) ([]byte, error) {
	if b, ok := v.([]byte); ok {
		return b, nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal outbox payload: %w", err)
	}
	return b, nil
}

// txOutboxRepo is the outbox half of a transactional TxRepository.
type txOutboxRepo struct {
	q *Queries
}

func (r *txOutboxRepo) CreateOutboxEvent(ctx context.Context, e *domain.OutboxEvent) error {
	uid, err := parseUUID(e.ID)
	if err != nil {
		return err
	}
	payload, err := marshalPayload(e.Payload)
	if err != nil {
		return err
	}
	return r.q.InsertOutboxEvent(ctx, InsertOutboxEventParams{
		ID:            uid,
		AggregateType: e.AggregateType,
		AggregateID:   e.AggregateID,
		EventType:     e.EventType,
		Payload:       payload,
	})
}

// txShipmentRepo is the shipment half of a transactional TxRepository.
type txShipmentRepo struct {
	q *Queries
}

func (r *txShipmentRepo) UpdateShipment(ctx context.Context, s *domain.Shipment) (*domain.Shipment, error) {
	uid, err := parseUUID(s.ID)
	if err != nil {
		return nil, err
	}
	row, err := r.q.UpdateShipment(ctx, UpdateShipmentParams{
		ID:             uid,
		TrackingNumber: s.TrackingNumber,
		DeliveryDate:   toTimestamptz(s.DeliveryDate),
		Status:         string(s.Status),
		Confirmed:      s.Confirmed,
		Name:           s.Address.Name,
		Street:         s.Address.Street,
		City:           s.Address.City,
		Country:        s.Address.Country,
		UpdatedAt:      toTimestamptz(s.UpdatedAt),
	})
	if err != nil {
		return nil, fmt.Errorf("tx update shipment: %w", err)
	}
	return toDomain(row), nil
}

// combinedTxRepo satisfies repository.TxRepository using a single pgx.Tx.
type combinedTxRepo struct {
	shipment *txShipmentRepo
	outbox   *txOutboxRepo
}

func (r *combinedTxRepo) UpdateShipment(ctx context.Context, s *domain.Shipment) (*domain.Shipment, error) {
	return r.shipment.UpdateShipment(ctx, s)
}

func (r *combinedTxRepo) CreateOutboxEvent(ctx context.Context, e *domain.OutboxEvent) error {
	return r.outbox.CreateOutboxEvent(ctx, e)
}

// newCombinedTxRepo builds a TxRepository backed by a tx-scoped Queries.
func newCombinedTxRepo(q *Queries) repository.TxRepository {
	return &combinedTxRepo{
		shipment: &txShipmentRepo{q: q},
		outbox:   &txOutboxRepo{q: q},
	}
}

// ensure pgtype.UUID is available for outbox_repo helpers
var _ pgtype.UUID
