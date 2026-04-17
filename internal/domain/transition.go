package domain

var allowedTransitions = map[DeliveryStatus][]DeliveryStatus{
	Created:         {Processing},
	Processing:      {Processed},
	Processed:       {DeliveryStarted},
	DeliveryStarted: {Delivered},
	Delivered:       {},
}

func IsValidTransition(from, to DeliveryStatus) bool {
	for _, allowed := range allowedTransitions[from] {
		if allowed == to {
			return true
		}
	}
	return false
}
