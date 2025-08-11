package payments

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
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
	if err := q.rdb.XGroupCreateMkStream(ctx, PaymentsStream, GroupName, "0").Err(); err != nil {
		if err.Error() != "BUSYGROUP Consumer Group name already exists" {
			return err
		}
	}

	return nil
}

func (q *PaymentsQueue) Enqueue(ctx context.Context, payment Payment) error {
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

func (q *PaymentsQueue) AckMessage(ctx context.Context, id string) error {
	return q.rdb.XAck(ctx, PaymentsStream, GroupName, id).Err()
}

func (q *PaymentsQueue) DeleteMessage(ctx context.Context, id string) error {
	return q.rdb.XDel(ctx, PaymentsStream, id).Err()
}

func (q *PaymentsQueue) AutoClaimPending(ctx context.Context, consumer string, minIdle time.Duration, start string, count int64) ([]redis.XMessage, string, error) {
	res := q.rdb.XAutoClaim(ctx, &redis.XAutoClaimArgs{
		Stream:   PaymentsStream,
		Group:    GroupName,
		Consumer: consumer,
		MinIdle:  minIdle,
		Start:    start,
		Count:    count,
	})
	if res.Err() != nil {
		return nil, start, res.Err()
	}
	messages, nextStart := res.Val()
	return messages, nextStart, nil
}
