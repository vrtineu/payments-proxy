# Payments Proxy - Rinha de Backend 2025

Uma solução em Go para intermediar pagamentos entre múltiplos processadores de pagamento, desenvolvida para a Rinha de Backend 2025.

## Arquitetura

```mermaid
graph TB
    %% Clients
    Client[Cliente/K6 Tests] --> HAProxy

    %% Load Balancer
    HAProxy[HAProxy Load Balancer<br/>:9999] --> App1
    HAProxy --> App2

    %% Application Instances
    subgraph "Application Layer"
        App1[App Instance 1<br/>:9999]
        App2[App Instance 2<br/>:9999]
    end

    %% Shared Infrastructure
    subgraph "Data Layer"
        Redis[(Redis<br/>Queue & Storage<br/>:6379)]
    end

    %% External Payment Processors
    subgraph "External Services"
        PaymentDefault[Payment Processor Default<br/>Taxa: 5%<br/>:8080]
        PaymentFallback[Payment Processor Fallback<br/>Taxa: 15%<br/>:8080]
    end

    %% Application connections
    App1 --> Redis
    App2 --> Redis
    App1 --> PaymentDefault
    App1 --> PaymentFallback
    App2 --> PaymentDefault
    App2 --> PaymentFallback

    %% Internal Components
    subgraph "App1 Components"
        Handler1[Payment Handlers]
        Queue1[Payments Queue<br/>Redis Streams]
        Storage1[Payments Storage<br/>Redis Sorted Sets]
        Worker1[Payment Workers<br/>x2-8 workers]
        Health1[Health Checker]
    end

    subgraph "App2 Components"
        Handler2[Payment Handlers]
        Queue2[Payments Queue<br/>Redis Streams]
        Storage2[Payments Storage<br/>Redis Sorted Sets]
        Worker2[Payment Workers<br/>x2-8 workers]
        Health2[Health Checker]
    end

    App1 -.-> Handler1
    App1 -.-> Queue1
    App1 -.-> Storage1
    App1 -.-> Worker1
    App1 -.-> Health1

    App2 -.-> Handler2
    App2 -.-> Queue2
    App2 -.-> Storage2
    App2 -.-> Worker2
    App2 -.-> Health2

    %% Data Flow
    Queue1 --> Redis
    Queue2 --> Redis
    Storage1 --> Redis
    Storage2 --> Redis
    Worker1 --> PaymentDefault
    Worker1 --> PaymentFallback
    Worker2 --> PaymentDefault
    Worker2 --> PaymentFallback
    Health1 --> PaymentDefault
    Health1 --> PaymentFallback
    Health2 --> PaymentDefault
    Health2 --> PaymentFallback

    %% Styling
    classDef external fill:#ffeb3b,stroke:#f57f17,stroke-width:2px
    classDef app fill:#4caf50,stroke:#2e7d32,stroke-width:2px
    classDef data fill:#2196f3,stroke:#1565c0,stroke-width:2px
    classDef lb fill:#ff9800,stroke:#ef6c00,stroke-width:2px
    classDef client fill:#9c27b0,stroke:#6a1b9a,stroke-width:2px

    class PaymentDefault,PaymentFallback external
    class App1,App2,Handler1,Handler2,Queue1,Queue2,Storage1,Storage2,Worker1,Worker2,Health1,Health2 app
    class Redis data
    class HAProxy lb
    class Client client
```

## Stack Tecnológica

- **Linguagem**: Go 1.24
- **Load Balancer**: HAProxy 3.1.7
- **Cache/Queue**: Redis 7.2
- **Containerização**: Docker & Docker Compose

## Endpoints

### Aplicação Principal

| Método | Endpoint | Descrição |
|--------|----------|-----------|
| `POST` | `/payments` | Processa um novo pagamento |
| `GET` | `/payments-summary` | Retorna resumo dos pagamentos processados |
| `GET` | `/health` | Health check da aplicação |

### Exemplo de Requisição

```bash
# Processar pagamento
curl -X POST http://localhost:9999/payments \
  -H "Content-Type: application/json" \
  -d '{
    "correlationId": "4a7901b8-7d26-4d9d-aa19-4dc1c7cf60b3",
    "amount": 19.90
  }'

# Consultar resumo
curl "http://localhost:9999/payments-summary?from=2025-01-01T00:00:00Z&to=2025-01-31T23:59:59Z"
```

## Como Executar

### 1. Clonar o Repositório
```bash
git clone https://github.com/vrtineu/payments-proxy.git
cd payments-proxy
```

### 2. Subir os Payment Processors
```bash
make start-gateways
```

### 3. Executar a Aplicação
```bash
# Desenvolvimento
make start

# "Produção" com Docker Compose
make compose
```

### 4. Executar Testes
```bash
make run-k6-tests
```

## Comandos Disponíveis

```bash
# Aplicação
make start              # Executa aplicação localmente
make build-prd          # Build para produção
make compose            # Sobe com Docker Compose
make compose-down       # Para containers

# Payment Gateways
make start-gateways     # Inicia gateways externos
make stop-gateways      # Para gateways
make down-gateways      # Remove gateways
make purge-gateways     # Limpa dados dos gateways

# Testes
make run-k6-tests       # Executa suite completa de testes
```

## Estratégia de Negócio

### Seleção de Gateway

1. **Verifica saúde** de ambos os gateways
2. **Prioriza gateway disponível** (Default primeiro)
3. **Escolhe por performance** quando ambos disponíveis (menor `minResponseTime`)
4. **Fallback automático** em caso de falha

### Processamento Assíncrono

1. **Requisições aceitas** imediatamente (HTTP 202)
2. **Enfileiramento** via Redis Streams
3. **Workers paralelos** processam fila
4. **Auto-claim** de mensagens orfãs
5. **Armazenamento** de resultados para auditoria

### Monitoramento de Saúde

- **Health checks** a cada 1 segundo
- **Rate limiting** respeitado (1 call / 5s por gateway)
- **Cache local** para reduzir latência
- **Distributed locking** via Redis para coordenação

## Resultados Esperados

- **Alta throughput**: Processamento de centenas de pagamentos/segundo
- **Baixa latência**: P99 < 10ms para máximo bônus
- **Zero inconsistências**: Auditoria rigorosa entre sistemas
- **Otimização de custos**: Maximização do uso do gateway com menor taxa

---

**Desenvolvido para a Rinha de Backend 2025** 🏆
