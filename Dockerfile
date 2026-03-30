# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o uvcb ./cmd/uvcb

# Runtime stage
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/uvcb .

CMD ["./uvcb"]
