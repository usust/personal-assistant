# syntax=docker/dockerfile:1

FROM golang:1.26.2-bookworm AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go test ./...
RUN CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o /out/personal-assistant .

FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates curl tzdata \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=builder /out/personal-assistant /app/personal-assistant

EXPOSE 16101
HEALTHCHECK --interval=5s --timeout=3s --start-period=10s --retries=6 \
  CMD curl --fail --silent http://127.0.0.1:16101/api/health >/dev/null || exit 1

ENTRYPOINT ["/app/personal-assistant"]
