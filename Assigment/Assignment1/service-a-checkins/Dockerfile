# ---------- Stage 1: build ----------
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src

COPY proto/ ./proto/

COPY service-a-checkins/go.mod  ./service-a-checkins/
COPY service-a-checkins/go.sum  ./service-a-checkins/
WORKDIR /src/service-a-checkins
RUN go mod edit -replace github.com/sangay/web303/proto=/src/proto && \
    go mod download

WORKDIR /src
COPY service-a-checkins/ ./service-a-checkins/

WORKDIR /src/service-a-checkins
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/checkins ./main.go

# ---------- Stage 2: runtime ----------
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --from=builder /out/checkins /app/checkins

EXPOSE 50052
USER nonroot:nonroot

ENTRYPOINT ["/app/checkins"]