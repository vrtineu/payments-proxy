package processor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

type HealthChecker struct {
	rdb             *redis.Client
	defaultGateway  *PaymentGateway
	fallbackGateway *PaymentGateway
	instanceID      string
}

type HealthStatus struct {
	Failing         bool  `json:"failing"`
	MinResponseTime int64 `json:"minResponseTime"`
}

func NewHealthChecker(rdb *redis.Client, defaultGateway *PaymentGateway, fallbackGateway *PaymentGateway) *HealthChecker {
	instanceID := os.Getenv("HOSTNAME")
	if instanceID == "" {
		instanceID = fmt.Sprintf("proc-%d", os.Getpid())
	}

	return &HealthChecker{
		rdb:             rdb,
		defaultGateway:  defaultGateway,
		fallbackGateway: fallbackGateway,
		instanceID:      instanceID,
	}
}

func (hc *HealthChecker) StartHealthMonitor(ctx context.Context) {
	ticker := time.NewTicker(6 * time.Second)

	go func() {
		for {
			select {
			case <-ticker.C:
				hc.DoHealthCheck(ctx, hc.defaultGateway)
				hc.DoHealthCheck(ctx, hc.fallbackGateway)
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
}

func (hc *HealthChecker) TryAcquireLock(ctx context.Context, paymentGateway *PaymentGateway) bool {
	key := fmt.Sprintf("health:check:%d:lock", paymentGateway.gatewayType)
	ok, err := hc.rdb.SetNX(ctx, key, hc.instanceID, 5*time.Second).Result() // Reduced to 5 seconds
	if err != nil {
		return false
	}
	return ok
}

func (hc *HealthChecker) DoHealthCheck(ctx context.Context, paymentGateway *PaymentGateway) {
	if !hc.TryAcquireLock(ctx, paymentGateway) {
		return
	}

	health, err := paymentGateway.HealthCheck(ctx)
	if err != nil {
		return
	}

	key := fmt.Sprintf("processor:%d:health", paymentGateway.gatewayType)
	err = hc.rdb.Set(ctx, key, health, 30*time.Second).Err()
	if err != nil {
		return
	}
}

func (hc *HealthChecker) GetHealthStatus(ctx context.Context, paymentGateway *PaymentGateway) (*HealthStatus, error) {
	key := fmt.Sprintf("processor:%d:health", paymentGateway.gatewayType)
	result := hc.rdb.Get(ctx, key)
	if result.Err() != nil {
		return nil, errors.New("health not cached")
	}

	var healthStatus HealthStatus
	if err := json.Unmarshal([]byte(result.Val()), &healthStatus); err != nil {
		return nil, err
	}
	return &healthStatus, nil
}
