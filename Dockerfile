FROM golang:1.26.1-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /build/app ./cmd/server

FROM alpine:3.20

WORKDIR /app
COPY --from=builder /build/app ./app

EXPOSE 8888

CMD ["./app"]
