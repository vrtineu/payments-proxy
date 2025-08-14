package main

import (
	"context"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

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
	go worker.Start(ctx)

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	http.HandleFunc("/payments", paymentHandlers.CreatePaymentHandler)
	http.HandleFunc("/payments-summary", paymentHandlers.PaymentsSummaryHandler)

	server := &http.Server{
		Addr:           ":9999",
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 16,
	}

	server.ListenAndServe()
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
