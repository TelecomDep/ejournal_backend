FROM golang:1.26.6-alpine3.24 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /build/app ./cmd/server

FROM alpine:3.24.1

WORKDIR /app
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /build/app ./app

RUN addgroup -S -g 10001 app && adduser -S -D -H -u 10001 -G app app \
    && chown -R app:app /app
USER 10001:10001

EXPOSE 8888

CMD ["./app"]
