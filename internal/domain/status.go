package domain

type DeliveryStatus string

const (
	Created          DeliveryStatus = "CREATED"
	Processing       DeliveryStatus = "PROCESSING"
	Processed        DeliveryStatus = "PROCESSED"
	DeliveryStarted  DeliveryStatus = "DELIVERY_STARTED"
	Delivered        DeliveryStatus = "DELIVERED"
)
