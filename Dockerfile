# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Копируем файлы модулей
COPY go.mod ./
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем приложение
RUN CGO_ENABLED=0 GOOS=linux go build -o /buildANDfly ./server

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Копируем бинарник
COPY --from=builder /buildANDfly .
# Копируем templates и migrations
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080

CMD ["./buildANDfly"]