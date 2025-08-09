package main

import (
	"context"
	"net/http"

	"github.com/vrtineu/payments-proxy/internal/infra/redis"
	"github.com/vrtineu/payments-proxy/internal/payments"
	"github.com/vrtineu/payments-proxy/internal/payments/processor"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	redisClient := redis.NewRedisClient()

	defaultGateway := processor.NewPaymentGateway("http://localhost:8001", payments.Default)
	fallbackGateway := processor.NewPaymentGateway("http://localhost:8002", payments.Fallback)

	healthChecker := processor.NewHealthChecker(redisClient.Client, defaultGateway, fallbackGateway)
	healthChecker.StartHealthMonitor(ctx)

	paymentsQueue := payments.NewPaymentsQueue(redisClient.Client)

	err := paymentsQueue.SetupPaymentsQueue(ctx)
	if err != nil {
		panic(err)
	}

	paymentHandlers := payments.NewPaymentHandlers(paymentsQueue)

	worker := processor.NewPaymentWorker(paymentsQueue, healthChecker, defaultGateway, fallbackGateway)
	go func() {
		worker.Start(ctx)
	}()

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	http.HandleFunc("/payments", paymentHandlers.CreatePaymentHandler)
	http.HandleFunc("/payments-summary", paymentHandlers.PaymentsSummaryHandler)
	http.ListenAndServe(":9999", nil)
}
