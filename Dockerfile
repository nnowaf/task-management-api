# ---- Build stage ----
FROM golang:1.24-alpine AS builder

WORKDIR /app
RUN apk add --no-cache git

# Cache dependencies first
COPY go.mod go.sum ./
RUN go mod download

# Build the static binary (docs/ is generated and committed, so the build needs no swag)
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# ---- Runtime stage ----
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 10001 appuser
WORKDIR /app
COPY --from=builder /out/api /app/api

USER appuser
EXPOSE 3000
ENTRYPOINT ["/app/api"]
