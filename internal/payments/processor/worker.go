package processor

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/vrtineu/payments-proxy/internal/payments"
)

type PaymentWorker struct {
	queue           *payments.PaymentsQueue
	healthChecker   *HealthChecker
	defaultGateway  *PaymentGateway
	fallbackGateway *PaymentGateway
}

func NewPaymentWorker(queue *payments.PaymentsQueue, healthChecker *HealthChecker, defaultGateway *PaymentGateway, fallbackGateway *PaymentGateway) *PaymentWorker {
	return &PaymentWorker{
		queue:           queue,
		healthChecker:   healthChecker,
		defaultGateway:  defaultGateway,
		fallbackGateway: fallbackGateway,
	}
}

func (pw *PaymentWorker) Start(ctx context.Context) {
	go func() {
		for {
			messages, err := pw.queue.Dequeue(ctx, pw.healthChecker.instanceID)
			if err != nil {
				fmt.Printf("Error dequeuing message: %v\n", err)
				continue
			}
			
			if len(messages) > 0 {
				for _, msg := range messages {
					pw.processMessage(msg)
				}
			}
		}
	}()
}

func (pw *PaymentWorker) processMessage(msg redis.XMessage) {
	fmt.Printf("Processing message: %v\n", msg)
}
