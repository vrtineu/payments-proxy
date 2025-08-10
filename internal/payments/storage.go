package payments

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type PaymentsStorage struct {
	rdb *redis.Client
}

func NewPaymentsStorage(rdb *redis.Client) *PaymentsStorage {
	return &PaymentsStorage{
		rdb: rdb,
	}
}

func (ps *PaymentsStorage) SaveToGatewaySets(ctx context.Context, payment *Payment) error {
	timestamp, err := time.Parse(time.RFC3339, payment.RequestedAt)
	if err != nil {
		return err
	}

	value := fmt.Sprintf("%s:%f", payment.CorrelationID, payment.Amount)
	key := fmt.Sprintf("payments:%s", payment.Gateway.String())

	return ps.rdb.ZAdd(ctx, key, redis.Z{
		Score:  float64(timestamp.UnixNano()),
		Member: value,
	}).Err()
}

func (ps *PaymentsStorage) GetPaymentsByScoreRange(ctx context.Context, gateway GatewayType, fromScore, toScore float64) ([]string, error) {
	key := fmt.Sprintf("payments:%s", gateway.String())
	return ps.rdb.ZRangeByScore(ctx, key, &redis.ZRangeBy{
		Min: fmt.Sprintf("%f", fromScore),
		Max: fmt.Sprintf("%f", toScore),
	}).Result()
}
