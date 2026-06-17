FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -X main.Version=docker -X main.BuildTime=$(date -u +%Y%m%d%H%M%S)" \
    -o beacon-server .

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /build/beacon-server .
COPY --from=builder /build/config/params.json ./config/params.json

EXPOSE 8080 6060

ENV GIN_MODE=release
ENV DEMO_MODE=false

ENTRYPOINT ["./beacon-server"]
