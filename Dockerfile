# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

# Собираем приложение из server
RUN CGO_ENABLED=0 GOOS=linux go build -o /buildANDfly ./server

# Final stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /buildANDfly .
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/migrations ./migrations
EXPOSE 8000
CMD ["./buildANDfly"]