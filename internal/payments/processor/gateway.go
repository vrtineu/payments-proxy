package processor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/vrtineu/payments-proxy/internal/payments"
)

type PaymentGateway struct {
	url         string
	gatewayType payments.GatewayType
	client      *http.Client
}

const (
	HealthCheckEndpoint        = "/payments/service-health"
	ProcessPaymentEndpoint     = "/payments"
	ServiceUnavailableResponse = `{"failing":true,"minResponseTime":0}`
)

func NewPaymentGateway(url string, gatewayType payments.GatewayType) *PaymentGateway {
	return &PaymentGateway{
		url:         url,
		gatewayType: gatewayType,
		client: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        256,
				MaxIdleConnsPerHost: 128,
				MaxConnsPerHost:     128,
				IdleConnTimeout:     30 * time.Second,
				DisableCompression:  true,
			},
		},
	}
}

func (pg *PaymentGateway) HealthCheck(ctx context.Context) ([]byte, error) {
	// Verifica as condições de funcionamento do endpoint de pagamentos. Limite de 1 chamada a cada 5 segundos.
	// GET /payments/service-health
	// HTTP 200 - Ok
	// {
	// 	"failing": false,
	// 	"minResponseTime": 100
	// }

	req, err := http.NewRequestWithContext(ctx, "GET", pg.url+HealthCheckEndpoint, nil)
	if err != nil {
		return []byte(ServiceUnavailableResponse), err
	}

	resp, err := pg.client.Do(req)
	if err != nil {
		return []byte(ServiceUnavailableResponse), err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []byte(ServiceUnavailableResponse), errors.New("health check failed: " + resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return []byte(ServiceUnavailableResponse), err
	}

	return body, nil
}

func (pg *PaymentGateway) ProcessPayment(ctx context.Context, payment *payments.Payment) error {
	// Requisita o processamento de um pagamento.
	// POST /payments
	// {
	// 	"correlationId": "4a7901b8-7d26-4d9d-aa19-4dc1c7cf60b3",
	// 	"amount": 19.90,
	// 	"requestedAt" : "2025-07-15T12:34:56.000Z"
	// }

	// HTTP 200 - Ok
	// {
	// 	"message": "payment processed successfully"
	// }

	paymentData, err := json.Marshal(payment)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", pg.url+ProcessPaymentEndpoint, bytes.NewBuffer(paymentData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := pg.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to process payment: %s, body: %s", resp.Status, body)
	}

	return nil
}
