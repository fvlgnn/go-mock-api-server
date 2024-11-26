# Fase 1: Build
FROM golang:alpine AS builder
WORKDIR /app
COPY main.go .
COPY go.* .
COPY ./config/ ./config/
RUN go build -o app .

# Fase 2: Runtime
FROM scratch
COPY --from=builder /app/app /app
COPY --from=builder /app/config/ /config/
ENTRYPOINT ["/app"]