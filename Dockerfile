# ---------------------------
# Build stage
# ---------------------------
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o server ./cmd/main.go

# ---------------------------
# Run stage
# ---------------------------
FROM alpine:latest
WORKDIR /app

RUN apk add --no-cache bash netcat-openbsd tzdata \
    && cp /usr/share/zoneinfo/Asia/Ho_Chi_Minh /etc/localtime \
    && echo "Asia/Ho_Chi_Minh" > /etc/timezone

COPY --from=builder /app/server .
COPY wait-for-it.sh .
RUN chmod +x wait-for-it.sh

# EXPOSE port cho Docker và Render
EXPOSE 8080

# Entry point: wait-for-it chỉ chờ DB nếu hostname là 'db'
# Khi deploy, DB_HOST sẽ là Managed DB, wait-for-it sẽ timeout 0 và chạy thẳng
ENTRYPOINT ["sh", "-c", "./wait-for-it.sh --host=${DB_HOST:-db} --port=${DB_PORT:-5432} -- ./server"]


