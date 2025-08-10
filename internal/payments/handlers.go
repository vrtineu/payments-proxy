package payments

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type PaymentHandlers struct {
	queue   *PaymentsQueue
	storage *PaymentsStorage
}

func NewPaymentHandlers(queue *PaymentsQueue, storage *PaymentsStorage) *PaymentHandlers {
	return &PaymentHandlers{
		queue:   queue,
		storage: storage,
	}
}

func (h *PaymentHandlers) CreatePaymentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		handleMethodNotAllowed(w)
		return
	}

	var payment Payment
	if err := json.NewDecoder(r.Body).Decode(&payment); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	h.queue.Enqueue(r.Context(), payment)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
}

type GatewaySummary struct {
	TotalRequests int64   `json:"totalRequests"`
	TotalAmount   float64 `json:"totalAmount"`
}

type PaymentsSummaryResponse struct {
	Default  GatewaySummary `json:"default"`
	Fallback GatewaySummary `json:"fallback"`
}

func (h *PaymentHandlers) PaymentsSummaryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handleMethodNotAllowed(w)
		return
	}

	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	fromTime, _ := time.Parse(time.RFC3339, fromStr)
	toTime, _ := time.Parse(time.RFC3339, toStr)
	fromScore := fromTime.UnixNano()
	toScore := toTime.UnixNano()

	defaultData, _ := h.storage.GetPaymentsByScoreRange(r.Context(), Default, float64(fromScore), float64(toScore))
	fallbackData, _ := h.storage.GetPaymentsByScoreRange(r.Context(), Fallback, float64(fromScore), float64(toScore))

	response := PaymentsSummaryResponse{
		Default:  h.calculateSummary(defaultData),
		Fallback: h.calculateSummary(fallbackData),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func handleMethodNotAllowed(w http.ResponseWriter) {
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (h *PaymentHandlers) calculateSummary(data []string) GatewaySummary {
	var totalAmount float64
	totalRequests := int64(len(data))

	for _, entry := range data {
		parts := strings.Split(entry, ":")
		if len(parts) == 2 {
			amount, _ := strconv.ParseFloat(parts[1], 64)
			totalAmount += amount
		}
	}

	return GatewaySummary{
		TotalRequests: totalRequests,
		TotalAmount:   totalAmount,
	}
}
