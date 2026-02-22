# Build Stage
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Build a statically linked binary
RUN CGO_ENABLED=0 go build -o main ./cmd/http

# Run Stage
FROM alpine:3.21
RUN apk --no-cache add ca-certificates
RUN adduser -D appuser
WORKDIR /home/appuser
COPY --from=builder --chown=appuser:appuser /app/main .
RUN chmod 500 main
USER appuser

EXPOSE 8080
CMD ["./main"]