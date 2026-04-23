package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"shipment_ms/internal/domain"
	"shipment_ms/internal/repository"
)

const (
	maxRetries    = 10
	batchSize     = 50
	minPollBackoff = 2 * time.Second
	maxPollBackoff = 60 * time.Second
)

// Publisher is the interface the worker needs from the Kafka producer.
// Satisfied by *kafka.KafkaProducer without importing that package here.
type Publisher interface {
	PublishRaw(ctx context.Context, topic, key string, payload []byte) error
}

// Worker polls the outbox table and publishes pending events to Kafka.
type Worker struct {
	outbox   repository.OutboxRepository
	producer Publisher
	interval time.Duration
	log      *slog.Logger
}

func NewWorker(
	outbox repository.OutboxRepository,
	producer Publisher,
	pollInterval time.Duration,
	log *slog.Logger,
) *Worker {
	return &Worker{
		outbox:   outbox,
		producer: producer,
		interval: pollInterval,
		log:      log,
	}
}

// Run starts the polling loop. It blocks until ctx is cancelled.
func (w *Worker) Run(ctx context.Context) {
	w.log.Info("outbox worker started", "poll_interval", w.interval)
	backoff := w.interval

	for {
		processed := w.processBatch(ctx)

		// If we processed a full batch there may be more — poll immediately.
		// Otherwise back off to reduce DB load during quiet periods.
		if processed == batchSize {
			backoff = minPollBackoff
		} else {
			backoff = min(backoff*2, maxPollBackoff)
			if backoff < minPollBackoff {
				backoff = minPollBackoff
			}
		}

		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			w.log.Info("outbox worker stopped")
			return
		case <-timer.C:
		}
	}
}

// processBatch fetches one batch of pending events and publishes each one.
// Returns the number of events processed.
func (w *Worker) processBatch(ctx context.Context) int {
	events, err := w.outbox.FetchPending(ctx, maxRetries, batchSize)
	if err != nil {
		w.log.Error("outbox: fetch pending failed", "error", err)
		return 0
	}

	for _, e := range events {
		w.publish(ctx, e)
	}
	return len(events)
}

func (w *Worker) publish(ctx context.Context, e *domain.OutboxEvent) {
	topic, key, err := routeEvent(e)
	if err != nil {
		w.log.Error("outbox: cannot route event",
			"event_id", e.ID,
			"event_type", e.EventType,
			"error", err,
		)
		_ = w.outbox.MarkFailed(ctx, e.ID)
		return
	}

	if err := w.producer.PublishRaw(ctx, topic, key, e.Payload); err != nil {
		w.log.Error("outbox: publish failed",
			"event_id", e.ID,
			"event_type", e.EventType,
			"aggregate_id", e.AggregateID,
			"retry_count", e.RetryCount,
			"error", err,
		)
		_ = w.outbox.MarkFailed(ctx, e.ID)
		return
	}

	if err := w.outbox.MarkSent(ctx, e.ID); err != nil {
		// The message was already published to Kafka. Log but don't retry
		// publishing — that would cause duplicates. The row stays FAILED and
		// will be retried, but the consumer must be idempotent (at-least-once).
		w.log.Error("outbox: mark sent failed after successful publish",
			"event_id", e.ID,
			"error", err,
		)
		return
	}

	w.log.Info("outbox: event published",
		"event_id", e.ID,
		"event_type", e.EventType,
		"topic", topic,
		"aggregate_id", e.AggregateID,
	)
}

// routeEvent maps an event type to its Kafka topic and partition key.
func routeEvent(e *domain.OutboxEvent) (topic, key string, err error) {
	switch e.EventType {
	case "delivery.confirmed":
		// Use shipment_id from payload as the Kafka key for consistent partitioning.
		key, err = extractShipmentID(e.Payload)
		if err != nil {
			return "", "", err
		}
		return "delivery.confirmed", key, nil
	default:
		return "", "", fmt.Errorf("unknown event type %q", e.EventType)
	}
}

func extractShipmentID(payload []byte) (string, error) {
	var v struct {
		ShipmentID string `json:"shipment_id"`
	}
	if err := json.Unmarshal(payload, &v); err != nil {
		return "", fmt.Errorf("extract shipment_id from payload: %w", err)
	}
	if v.ShipmentID == "" {
		return "", fmt.Errorf("shipment_id missing from payload")
	}
	return v.ShipmentID, nil
}

