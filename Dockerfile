FROM golang:1.19-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# COPY .env .env


RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o tester ./cmd/app
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/tester .
# COPY --from=builder /app/.env .env

EXPOSE 8080

CMD ["./tester"]
