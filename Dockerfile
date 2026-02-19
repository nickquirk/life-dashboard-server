# Build Stage
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Build the Go binary
RUN go build -o main ./cmd/http

# Run Stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .
COPY --from=builder /app/.env . 

EXPOSE 8080
CMD ["./main"]