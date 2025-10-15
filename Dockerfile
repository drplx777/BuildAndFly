# Build stage
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o BuildAndFly ./cmd/server

# Runtime stage
FROM alpine:latest AS runner
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app

COPY --from=builder /app/go.mod .
COPY --from=builder /app/go.sum .
COPY --from=builder /app/tasker .

EXPOSE 8000
CMD ["./BFA"]
