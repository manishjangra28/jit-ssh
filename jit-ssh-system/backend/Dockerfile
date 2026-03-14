# syntax=docker/dockerfile:1
FROM golang:1.24-alpine AS builder

# Ensure we use the toolchain matching go.mod
ENV GOTOOLCHAIN=auto

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o main main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/main .
COPY --from=builder /app/bin ./bin

EXPOSE 8080
CMD ["./main"]
