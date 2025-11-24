# 1️⃣ Build stage
FROM golang:1.24 AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/api

# 2️⃣ Run stage
FROM alpine:latest
WORKDIR /app

COPY --from=builder /app/main .

EXPOSE 8080
CMD ["./main"]
