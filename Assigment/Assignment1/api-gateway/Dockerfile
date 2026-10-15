# ---------- Stage 1: build ----------
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src

COPY proto/ ./proto/

COPY api-gateway/go.mod  ./api-gateway/
COPY api-gateway/go.sum  ./api-gateway/
WORKDIR /src/api-gateway
RUN go mod edit -replace github.com/sangay/web303/proto=/src/proto && \
    go mod download

WORKDIR /src
COPY api-gateway/ ./api-gateway/

WORKDIR /src/api-gateway
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/gateway ./main.go

# ---------- Stage 2: runtime ----------
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --from=builder /out/gateway /app/gateway

EXPOSE 8080
USER nonroot:nonroot

ENTRYPOINT ["/app/gateway"]