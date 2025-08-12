# Build stage
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache make

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN make build-prd

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates curl

WORKDIR /app
COPY --from=builder /app/server .

EXPOSE 9999

CMD ["./server"]
