FROM golang:1.26.6-alpine3.24 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o postgres-exporter cmd/postgres-exporter/main.go

FROM alpine:3.24.1
WORKDIR /app
COPY --from=builder /app/postgres-exporter .
RUN addgroup -S -g 10001 app && adduser -S -D -H -u 10001 -G app app \
    && chown -R app:app /app
USER 10001:10001
CMD ["./postgres-exporter"]
