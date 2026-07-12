FROM golang:1.26.1-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o postgres-exporter cmd/postgres-exporter/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/postgres-exporter .
CMD ["./postgres-exporter"]
