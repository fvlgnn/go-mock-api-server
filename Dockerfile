# syntax=docker/dockerfile:1

FROM golang:1.27.1-alpine AS builder

WORKDIR /src
COPY go.mod ./
COPY *.go ./

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

RUN CGO_ENABLED=0 go build \
    -trimpath \
    -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildDate=${BUILD_DATE}" \
    -o /go-mock-api-server .

FROM scratch

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

LABEL org.opencontainers.image.title="go-mock-api-server" \
      org.opencontainers.image.description="Configurable mock HTTP API server" \
      org.opencontainers.image.source="https://github.com/fvlgnn/go-mock-api-server" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${COMMIT}" \
      org.opencontainers.image.created="${BUILD_DATE}" \
      org.opencontainers.image.licenses="MIT"

COPY --from=builder /go-mock-api-server /go-mock-api-server
COPY config/ /config/

ENV CONFIG_DIR=/config \
    SERVER_PORT=8080

USER 65532:65532
EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=3s --start-period=2s --retries=3 \
    CMD ["/go-mock-api-server", "healthcheck"]

ENTRYPOINT ["/go-mock-api-server"]
