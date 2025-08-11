package payments

import (
	"time"
)

type GatewayType int

const (
	Default GatewayType = iota
	Fallback
)

type Payment struct {
	CorrelationID string      `json:"correlationId"`
	Amount        float64     `json:"amount"`
	RequestedAt   string      `json:"requestedAt"`
	Gateway       GatewayType `json:"-"`
}

func NewPayment(correlationID string, amount float64, gateway GatewayType) *Payment {
	return &Payment{
		CorrelationID: correlationID,
		Amount:        amount,
		RequestedAt:   time.Now().UTC().Format(time.RFC3339),
		Gateway:       gateway,
	}
}

func (g GatewayType) String() string {
	switch g {
	case Default:
		return "default"
	case Fallback:
		return "fallback"
	default:
		return "unknown"
	}
}
