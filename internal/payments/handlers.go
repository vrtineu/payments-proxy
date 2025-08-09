package payments

import (
	"encoding/json"
	"net/http"
)

type PaymentHandlers struct {
	queue *PaymentsQueue
}

func NewPaymentHandlers(queue *PaymentsQueue) *PaymentHandlers {
	return &PaymentHandlers{
		queue: queue,
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

func (h *PaymentHandlers) PaymentsSummaryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handleMethodNotAllowed(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

func handleMethodNotAllowed(w http.ResponseWriter) {
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}
