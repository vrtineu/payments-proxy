package payments

import (
	"context"

	"github.com/redis/go-redis/v9"
	"github.com/vrtineu/payments-proxy/internal/payments/entities"
)

const (
	PaymentsStream = "payments"
)

type PaymentsQueue struct {
	rdb *redis.Client
}

func NewPaymentsQueue(rdb *redis.Client) *PaymentsQueue {
	return &PaymentsQueue{
		rdb: rdb,
	}
}

func (q *PaymentsQueue) Enqueue(ctx context.Context, payment entities.Payment) {
	q.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: PaymentsStream,
		Values: map[string]any{
			"correlationId": payment.CorrelationID,
			"amount":        payment.Amount,
		},
	})
}
