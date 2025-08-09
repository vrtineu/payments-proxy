package processor

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

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
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		pw.runDequeueWorker(ctx)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		pw.runAutoClaimWorker(ctx)
	}()

	<-ctx.Done()

	wg.Wait()
}

func (pw *PaymentWorker) runDequeueWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if err := pw.handleNormalMessages(ctx); err != nil {
				fmt.Printf("Error in dequeue worker: %v\n", err)
			}
		}
	}
}

func (pw *PaymentWorker) runAutoClaimWorker(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	start := "0-0"

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			nextStart, err := pw.handleAutoClaimMessages(ctx, start)
			if err != nil {
				fmt.Printf("Error in auto claim worker: %v\n", err)
				start = "0-0"
				continue
			}
			start = nextStart
		}
	}
}

func (pw *PaymentWorker) handleNormalMessages(ctx context.Context) error {
	messages, err := pw.queue.Dequeue(ctx, pw.healthChecker.instanceID)
	if err != nil {
		return fmt.Errorf("error dequeuing messages: %w", err)
	}
	if len(messages) == 0 {
		return nil
	}

	for _, msg := range messages {
		pw.processMessage(ctx, msg)
	}

	return nil
}

func (pw *PaymentWorker) handleAutoClaimMessages(ctx context.Context, start string) (string, error) {
	messages, nextStart, err := pw.queue.AutoClaimPending(
		ctx,
		pw.healthChecker.instanceID,
		10*time.Second,
		start,
		10,
	)
	if err != nil {
		return "0-0", fmt.Errorf("auto-claim failed: %w", err)
	}

	if len(messages) == 0 {
		return "0-0", nil
	}

	for _, msg := range messages {
		pw.processMessage(ctx, msg)
	}

	return nextStart, nil
}

func (pw *PaymentWorker) processMessage(ctx context.Context, msg redis.XMessage) {
	correlationID, amount, err := pw.parseMessageData(msg)
	if err != nil {
		return
	}

	gateway := pw.getPaymentGateway(ctx)
	if gateway == nil {
		return
	}

	payment := payments.NewPayment(correlationID, amount, gateway.gatewayType)

	if err := gateway.ProcessPayment(ctx, payment); err != nil {
		return
	}

	pw.handleMessageCompletion(ctx, msg.ID)
}

func (pw *PaymentWorker) parseMessageData(msg redis.XMessage) (string, float64, error) {
	correlationID, ok := msg.Values["correlationId"].(string)
	if !ok {
		return "", 0, fmt.Errorf("correlationId not found or invalid type")
	}

	amountStr, ok := msg.Values["amount"].(string)
	if !ok {
		return correlationID, 0, fmt.Errorf("amount not found or invalid type")
	}

	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return correlationID, 0, fmt.Errorf("invalid amount format: %w", err)
	}

	return correlationID, amount, nil
}

func (pw *PaymentWorker) handleMessageCompletion(ctx context.Context, messageID string) {
	if err := pw.queue.AckMessage(ctx, messageID); err != nil {
		fmt.Printf("Error acknowledging message %s: %v\n", messageID, err)
		return
	}

	if err := pw.queue.DeleteMessage(ctx, messageID); err != nil {
		fmt.Printf("Error deleting message %s: %v\n", messageID, err)
	}
}

func (pw *PaymentWorker) getPaymentGateway(ctx context.Context) *PaymentGateway {
	defaultStatus, _ := pw.healthChecker.GetHealthStatus(ctx, pw.defaultGateway)
	fallbackStatus, _ := pw.healthChecker.GetHealthStatus(ctx, pw.fallbackGateway)

	if !defaultStatus.Failing && defaultStatus.MinResponseTime < 1000 {
		return pw.defaultGateway
	} else if !fallbackStatus.Failing {
		return pw.fallbackGateway
	}

	return nil
}
