package main

import (
	"net/http"

	"github.com/vrtineu/payments-proxy/internal/infra/redis"
	"github.com/vrtineu/payments-proxy/internal/payments"
)

func main() {
	redisClient := redis.NewRedisClient()
	paymentsQueue := payments.NewPaymentsQueue(redisClient.Client)
	paymentHandlers := payments.NewPaymentHandlers(paymentsQueue)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	http.HandleFunc("/payments", paymentHandlers.CreatePaymentHandler)
	http.HandleFunc("/payments-summary", paymentHandlers.PaymentsSummaryHandler)
	http.ListenAndServe(":9999", nil)
}
