package entities

type Payment struct {
	CorrelationID string  `json:"correlationId"`
	Amount        float64 `json:"amount"`
}
