package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	kafka "github.com/segmentio/kafka-go"

	"shipment_ms/internal/domain"
)

const topicDeliveryConfirmed = "delivery.confirmed"

// message is the JSON shape written to Kafka.
type message struct {
	ShipmentID     string    `json:"shipment_id"`
	OrderID        string    `json:"order_id"`
	TrackingNumber string    `json:"tracking_number"`
	DeliveredAt    time.Time `json:"delivered_at"`
}

type KafkaProducer struct {
	writer *kafka.Writer
	log    *slog.Logger
}

// NewKafkaProducer creates a producer that writes to the delivery.confirmed
// topic. brokers is a comma-separated list, e.g. "localhost:9092".
func NewKafkaProducer(brokers []string, log *slog.Logger) *KafkaProducer {
	w := &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Topic:                  topicDeliveryConfirmed,
		Balancer:               &kafka.Hash{}, // routes by key → consistent partition per shipment
		RequiredAcks:           kafka.RequireAll,
		AllowAutoTopicCreation: true,
	}
	return &KafkaProducer{writer: w, log: log}
}

// PublishDeliveryConfirmed satisfies usecase.EventPublisher.
func (p *KafkaProducer) PublishDeliveryConfirmed(ctx context.Context, event domain.DeliveryConfirmedEvent) error {
	payload, err := json.Marshal(message{
		ShipmentID:     event.ShipmentID,
		OrderID:        event.OrderID,
		TrackingNumber: event.TrackingNumber,
		DeliveredAt:    event.DeliveredAt,
	})
	if err != nil {
		return fmt.Errorf("marshal delivery confirmed event: %w", err)
	}

	err = p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(event.ShipmentID), // partition key — all events for a shipment go to the same partition
		Value: payload,
	})
	if err != nil {
		return fmt.Errorf("write delivery.confirmed to kafka: %w", err)
	}

	p.log.InfoContext(ctx, "kafka event published",
		"topic", topicDeliveryConfirmed,
		"shipment_id", event.ShipmentID,
		"order_id", event.OrderID,
	)
	return nil
}

// PublishRaw writes a raw payload to any topic with the given key.
// Used by the outbox worker to replay persisted events.
func (p *KafkaProducer) PublishRaw(ctx context.Context, topic, key string, payload []byte) error {
	w := &kafka.Writer{
		Addr:         p.writer.Addr,
		Topic:        topic,
		Balancer:     &kafka.Hash{},
		RequiredAcks: kafka.RequireAll,
	}
	defer w.Close()
	err := w.WriteMessages(ctx, kafka.Message{
		Key:   []byte(key),
		Value: payload,
	})
	if err != nil {
		return fmt.Errorf("write raw message to topic %q: %w", topic, err)
	}
	return nil
}

// Close flushes pending messages and releases the writer.
func (p *KafkaProducer) Close() error {
	return p.writer.Close()
}
