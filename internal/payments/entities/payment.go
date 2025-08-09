package entities

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
		RequestedAt:   time.Now().UTC().Format(time.RFC3339Nano),
		Gateway:       gateway,
	}
}
