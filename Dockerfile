# Build stage
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o bfa ./cmd/server

# Runtime stage
FROM alpine:latest AS runner
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app

COPY --from=builder /app/go.mod /app/go.mod
COPY --from=builder /app/go.sum /app.go.sum

COPY --from=builder /app/bfa /app/bfa
COPY --from=builder /app/pkg/templates /app/pkg/templates

EXPOSE 8000
CMD ["./bfa"]
