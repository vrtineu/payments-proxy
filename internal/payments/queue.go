package payments

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/vrtineu/payments-proxy/internal/payments/entities"
)

const (
	PaymentsStream = "payments_stream"
	GroupName      = "payments"
)

type PaymentsQueue struct {
	rdb *redis.Client
}

func NewPaymentsQueue(rdb *redis.Client) *PaymentsQueue {
	return &PaymentsQueue{
		rdb: rdb,
	}
}

func (q *PaymentsQueue) SetupPaymentsQueue(ctx context.Context) error {
	err := q.rdb.Del(ctx, PaymentsStream).Err()
	if err != nil {
		return err
	}

	err = q.rdb.XGroupCreateMkStream(ctx, PaymentsStream, GroupName, "0").Err()
	if err != nil {
		return err
	}

	return nil
}

func (q *PaymentsQueue) Enqueue(ctx context.Context, payment entities.Payment) error {
	err := q.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: PaymentsStream,
		Values: map[string]any{
			"correlationId": payment.CorrelationID,
			"amount":        payment.Amount,
		},
	}).Err()

	return err
}

func (q *PaymentsQueue) Dequeue(ctx context.Context, instanceID string) ([]redis.XMessage, error) {
	result := q.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    GroupName,
		Consumer: instanceID,
		Streams:  []string{PaymentsStream, ">"},
		Count:    1,
		Block:    0,
	})

	if result.Err() != nil {
		fmt.Printf("Error reading from stream %s: %v\n", PaymentsStream, result.Err())
		return nil, result.Err()
	}

	streams := result.Val()
	if len(streams) == 0 {
		return nil, nil
	}

	stream := streams[0]
	return stream.Messages, nil
}
