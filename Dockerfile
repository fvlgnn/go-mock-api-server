# Fase 1: Build
FROM golang:alpine AS builder
ARG CONFIG_DIR=config
WORKDIR /app
COPY main.go .
COPY go.* .
COPY ./${CONFIG_DIR}/ ./config/
RUN go build -o app .

# Fase 2: Runtime
FROM scratch
COPY --from=builder /app/app /app
COPY --from=builder /app/config/ /config/
ENTRYPOINT ["/app"]