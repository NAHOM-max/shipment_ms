package domain

import "time"

type OutboxStatus string

const (
	OutboxPending OutboxStatus = "PENDING"
	OutboxSent    OutboxStatus = "SENT"
	OutboxFailed  OutboxStatus = "FAILED"
)

type OutboxEvent struct {
	ID            string
	AggregateType string
	AggregateID   string
	EventType     string
	Payload       []byte
	Status        OutboxStatus
	RetryCount    int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
