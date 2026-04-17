package domain

import "fmt"

var allowedTransitions = map[DeliveryStatus][]DeliveryStatus{
	Created:         {Processing},
	Processing:      {Processed},
	Processed:       {DeliveryStarted},
	DeliveryStarted: {Delivered},
	Delivered:       {},
}

// ErrInvalidTransition is returned when a requested status change violates
// the allowed state machine.
var ErrInvalidTransition = fmt.Errorf("invalid status transition")

func IsValidTransition(from, to DeliveryStatus) bool {
	for _, allowed := range allowedTransitions[from] {
		if allowed == to {
			return true
		}
	}
	return false
}
