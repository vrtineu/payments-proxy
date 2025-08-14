DEFAULT_PROCESSOR_URL=http://localhost:8001
FALLBACK_PROCESSOR_URL=http://localhost:8002
RINHA_TOKEN=123

# App commands

.PHONY: start build-prd compose compose-down

start:
	@echo "Starting payment processor..."
	@go run ./cmd/server/server.go

build-prd:
	@echo "Building payment processor for production..."
	@CGO_ENABLED=0 GOOS=linux go build -o server cmd/server/server.go

compose:
	@echo "Starting payment processor with Docker Compose..."
	@docker compose up -d --build

compose-down:
	@echo "Stopping payment processor with Docker Compose..."
	@docker compose down

# Gateways commands

.PHONY: start-gateways stop-gateways down-gateways purge-gateways

start-gateways:
	@echo "Starting payment gateways..."
	@cd misc/payment-processor && docker compose up -d

stop-gateways:
	@echo "Stopping payment gateways..."
	@cd misc/payment-processor && docker compose stop

down-gateways:
	@echo "Stopping payment gateways..."
	@cd misc/payment-processor && docker compose down

purge-gateways:
	@echo "Purging payment data..."
	@default_response=$$(curl -s -X POST $(DEFAULT_PROCESSOR_URL)/admin/purge-payments \
		-H "Content-Type: application/json" \
		-H "X-Rinha-Token: $(RINHA_TOKEN)"); \
	fallback_response=$$(curl -s -X POST $(FALLBACK_PROCESSOR_URL)/admin/purge-payments \
		-H "Content-Type: application/json" \
		-H "X-Rinha-Token: $(RINHA_TOKEN)"); \
	echo "Default response: $$default_response"; \
	echo "Fallback response: $$fallback_response"

# K6 testes

.PHONY: run-k6-tests

run-k6-tests:
	@echo "Running K6 tests..."
	@make purge-gateways
	@make compose-down
	@make compose
	@cd misc/rinha-test && k6 run rinha.js && cd -
