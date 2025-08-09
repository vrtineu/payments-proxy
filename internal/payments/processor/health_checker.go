package processor

import (
	"context"
	"encoding/json"
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

func (hc *HealthChecker) GetHealthStatus(ctx context.Context, paymentGateway *PaymentGateway) (*HealthStatus, error) {
	key := fmt.Sprintf("processor:%d:health", paymentGateway.gatewayType)
	result := hc.rdb.Get(ctx, key)
	if result.Err() != nil {
		return &HealthStatus{Failing: true, MinResponseTime: 0}, nil
	}

	var healthStatus HealthStatus
	if err := json.Unmarshal([]byte(result.Val()), &healthStatus); err != nil {
		return nil, err
	}
	return &healthStatus, nil
}

func (hc *HealthChecker) StartHealthMonitor(ctx context.Context) {
	hc.checkHealth(ctx)
	ticker := time.NewTicker(6 * time.Second)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				hc.checkHealth(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (hc *HealthChecker) checkHealth(ctx context.Context) {
	hc.doHealthCheck(ctx, hc.defaultGateway)
	hc.doHealthCheck(ctx, hc.fallbackGateway)
}

func (hc *HealthChecker) tryAcquireLock(ctx context.Context, paymentGateway *PaymentGateway) bool {
	key := fmt.Sprintf("health:check:%d:lock", paymentGateway.gatewayType)
	ok, err := hc.rdb.SetNX(ctx, key, hc.instanceID, 5*time.Second).Result()
	if err != nil {
		return false
	}
	return ok
}

func (hc *HealthChecker) doHealthCheck(ctx context.Context, paymentGateway *PaymentGateway) {
	if !hc.tryAcquireLock(ctx, paymentGateway) {
		return
	}

	healthBytes, _ := paymentGateway.HealthCheck(ctx)
	key := fmt.Sprintf("processor:%d:health", paymentGateway.gatewayType)
	_ = hc.rdb.Set(ctx, key, healthBytes, 30*time.Second).Err()
}
