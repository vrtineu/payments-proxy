package main

import (
	"context"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"

	"github.com/vrtineu/payments-proxy/internal/infra/redis"
	"github.com/vrtineu/payments-proxy/internal/payments"
	"github.com/vrtineu/payments-proxy/internal/payments/processor"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if os.Getenv("ENABLE_PPROF") == "true" {
		go func() {
			http.ListenAndServe(":6060", nil)
		}()
	}

	redisClient := redis.NewRedisClient()

	defaultGatewayUrl, fallbackGatewayUrl := getGatewayUrls()
	defaultGateway := processor.NewPaymentGateway(defaultGatewayUrl, payments.Default)
	fallbackGateway := processor.NewPaymentGateway(fallbackGatewayUrl, payments.Fallback)

	healthChecker := processor.NewHealthChecker(
		redisClient.Client,
		defaultGateway,
		fallbackGateway,
	)
	go healthChecker.StartHealthMonitor(ctx)

	paymentsQueue := payments.NewPaymentsQueue(redisClient.Client)

	err := paymentsQueue.SetupPaymentsQueue(ctx)
	if err != nil {
		panic(err)
	}

	paymentsStorage := payments.NewPaymentsStorage(redisClient.Client)
	paymentHandlers := payments.NewPaymentHandlers(paymentsQueue, paymentsStorage)

	worker := processor.NewPaymentWorker(
		paymentsQueue,
		paymentsStorage,
		healthChecker,
		defaultGateway,
		fallbackGateway,
	)

	numWorkers := runtime.NumCPU()
	if numWorkers < 2 {
		numWorkers = 2
	}
	if numWorkers > 8 {
		numWorkers = 8
	}

	for i := 0; i < numWorkers; i++ {
		go worker.Start(ctx)
	}

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	http.HandleFunc("/payments", paymentHandlers.CreatePaymentHandler)
	http.HandleFunc("/payments-summary", paymentHandlers.PaymentsSummaryHandler)

	http.ListenAndServe(":9999", nil)
}

func getGatewayUrls() (defaultGatewayUrl, fallbackGatewayUrl string) {
	defaultGatewayUrl = os.Getenv("DEFAULT_GATEWAY_URL")
	if defaultGatewayUrl == "" {
		defaultGatewayUrl = "http://localhost:8001"
	}

	fallbackGatewayUrl = os.Getenv("FALLBACK_GATEWAY_URL")
	if fallbackGatewayUrl == "" {
		fallbackGatewayUrl = "http://localhost:8082"
	}

	return
}
