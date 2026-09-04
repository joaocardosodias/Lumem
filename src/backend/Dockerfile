FROM golang:1.26-alpine3.21 AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /app/bin/api cmd/api/main.go

FROM alpine:3.21

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata curl && \
    adduser -D -u 10001 appuser

COPY --from=builder /app/bin/api /app/api
COPY --from=builder /app/migrations /app/migrations
COPY --from=builder /app/docs/openapi.json /app/docs/openapi.json

RUN chown -R appuser:appuser /app

USER appuser

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/healthz/live || exit 1

ENTRYPOINT ["/app/api"]
