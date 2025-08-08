package entities

import "time"

type GatewayType int

const (
	Default GatewayType = iota
	Fallback
)

type Payment struct {
	CorrelationID string      `json:"correlationId"`
	Amount        float64     `json:"amount"`
	RequestedAt   time.Time   `json:"requestedAt"`
	Gateway       GatewayType `json:"-"`
}
