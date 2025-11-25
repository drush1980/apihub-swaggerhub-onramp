FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /plugin

FROM alpine:3.18

COPY --from=builder /plugin /plugin

CMD ["/plugin"]
