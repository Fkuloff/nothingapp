FROM golang:1.25-alpine AS builder

WORKDIR /app

# 1. Сначала только модули
COPY . .
RUN go mod tidy


RUN go mod download

# 2. Потом весь проект


# 3. Собираем именно cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -o messenger ./cmd/server

# ===================== финальный образ =====================
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root

COPY --from=builder /app/messenger .

EXPOSE 8080

CMD ["./messenger"]
