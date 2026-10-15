# ---------- Stage 1: build ----------
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src

COPY proto/ ./proto/

COPY service-b-members/go.mod      ./service-b-members/
COPY service-b-members/go.sum      ./service-b-members/
WORKDIR /src/service-b-members
RUN go mod edit -replace github.com/sangay/web303/proto=/src/proto && \
    go mod download

WORKDIR /src
COPY service-b-members/ ./service-b-members/

WORKDIR /src/service-b-members
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/members ./main.go

# ---------- Stage 2: runtime ----------
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --from=builder /out/members /app/members

EXPOSE 50051
USER nonroot:nonroot

ENTRYPOINT ["/app/members"]
