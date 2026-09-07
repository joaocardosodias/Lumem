# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.26
ARG ALPINE_VERSION=3.23

FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS builder

WORKDIR /src

RUN apk add --no-cache ca-certificates git

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY cmd ./cmd
COPY internal ./internal

ARG TARGETOS=linux
ARG TARGETARCH=amd64
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -buildvcs=false -tags netgo,osusergo \
    -ldflags="-s -w -buildid=" -o /out/api ./cmd/api && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -buildvcs=false -tags netgo,osusergo \
    -ldflags="-s -w -buildid=" -o /out/worker ./cmd/worker && \
    install -d -m 1777 /out/rootfs/tmp

FROM scratch AS runtime

WORKDIR /app
USER 10001:10001

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /out/rootfs/ /

FROM runtime AS api

COPY --from=builder /out/api /app/api
COPY migrations /app/migrations
COPY docs/openapi.json /app/docs/openapi.json

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD ["/app/api", "healthcheck"]

ENTRYPOINT ["/app/api"]

FROM runtime AS worker

COPY --from=builder /out/worker /app/worker

ENTRYPOINT ["/app/worker"]

# Mantém `docker build .` compatível, produzindo a imagem da API por padrão.
FROM api AS final
