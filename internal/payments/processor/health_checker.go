package processor

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/vrtineu/payments-proxy/internal/payments"
)

type HealthChecker struct {
	rdb             *redis.Client
	instanceID      string
	localCache      map[payments.GatewayType]*HealthStatus
	lastUpdate      map[payments.GatewayType]time.Time
	mu              sync.RWMutex
	defaultGateway  *PaymentGateway
	fallbackGateway *PaymentGateway
}

type HealthStatus struct {
	Failing         bool  `json:"failing"`
	MinResponseTime int64 `json:"minResponseTime"`
}

func NewHealthChecker(rdb *redis.Client, defaultGateway, fallbackGateway *PaymentGateway) *HealthChecker {
	instanceID := os.Getenv("HOSTNAME")
	if instanceID == "" {
		instanceID = fmt.Sprintf("proc-%d", os.Getpid())
	}

	return &HealthChecker{
		rdb:             rdb,
		instanceID:      instanceID,
		localCache:      make(map[payments.GatewayType]*HealthStatus),
		lastUpdate:      make(map[payments.GatewayType]time.Time),
		defaultGateway:  defaultGateway,
		fallbackGateway: fallbackGateway,
	}
}

func (hc *HealthChecker) GetHealthStatus(ctx context.Context, gateway *PaymentGateway) (*HealthStatus, error) {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	status, exists := hc.localCache[gateway.gatewayType]
	if !exists {
		return &HealthStatus{Failing: true, MinResponseTime: 0}, nil
	}

	return status, nil
}

func (hc *HealthChecker) StartHealthMonitor(ctx context.Context) {
	hc.initializeCache()

	ticker := time.NewTicker(1 * time.Second)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				hc.checkAndUpdateHealth(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (hc *HealthChecker) initializeCache() {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	hc.localCache[payments.Default] = &HealthStatus{Failing: true, MinResponseTime: 0}
	hc.localCache[payments.Fallback] = &HealthStatus{Failing: true, MinResponseTime: 0}
}

func (hc *HealthChecker) checkAndUpdateHealth(ctx context.Context) {
	gateways := []payments.GatewayType{payments.Default, payments.Fallback}

	for _, gw := range gateways {
		if hc.shouldPerformHealthCheck(ctx, gw) {
			go hc.performHealthCheckWithLease(ctx, gw)
		}

		hc.refreshLocalCache(ctx, gw)
	}
}

func (hc *HealthChecker) shouldPerformHealthCheck(ctx context.Context, gateway payments.GatewayType) bool {
	leaseKey := fmt.Sprintf("health:lease:%s", gateway.String())

	acquired, err := hc.rdb.SetNX(ctx, leaseKey, hc.instanceID, 6*time.Second).Result()
	if err != nil {
		log.Printf("Error acquiring lease for %s: %v", gateway.String(), err)
		return false
	}

	return acquired
}

func (hc *HealthChecker) performHealthCheckWithLease(ctx context.Context, gateway payments.GatewayType) {
	var pg *PaymentGateway
	if gateway == payments.Default {
		pg = hc.defaultGateway
	} else {
		pg = hc.fallbackGateway
	}

	healthBytes, err := pg.HealthCheck(ctx)

	key := fmt.Sprintf("processor:%s:health", gateway.String())

	if err == nil {
		if setErr := hc.rdb.Set(ctx, key, healthBytes, 15*time.Second).Err(); setErr != nil {
			log.Printf("Error saving health status for %s: %v", gateway.String(), setErr)
		}
		hc.updateLocalCacheFromBytes(gateway, healthBytes)
	} else {
		log.Printf("Health check failed for %s: %v", gateway.String(), err)
		failingStatus := `{"failing":true,"minResponseTime":0}`
		hc.rdb.Set(ctx, key, failingStatus, 15*time.Second)
		hc.updateLocalCacheFromBytes(gateway, []byte(failingStatus))
	}
}

func (hc *HealthChecker) refreshLocalCache(ctx context.Context, gateway payments.GatewayType) {
	key := fmt.Sprintf("processor:%s:health", gateway.String())

	val, err := hc.rdb.Get(ctx, key).Result()
	if err == nil {
		hc.updateLocalCacheFromBytes(gateway, []byte(val))
	} else if err != redis.Nil {
		log.Printf("Error refreshing cache for %s: %v", gateway.String(), err)
	}
}

func (hc *HealthChecker) updateLocalCacheFromBytes(gateway payments.GatewayType, data []byte) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	status := &HealthStatus{}
	if err := json.Unmarshal(data, status); err != nil {
		log.Printf("Error unmarshaling health status for %s: %v", gateway.String(), err)
		status.Failing = true
		status.MinResponseTime = 0
	}

	hc.localCache[gateway] = status
	hc.lastUpdate[gateway] = time.Now()
}
