# ---------------------------------------------------------------------------
# Stage 1 — builder
# ---------------------------------------------------------------------------
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Cache dependencies first
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/anti-bruteforce ./cmd/server

# ---------------------------------------------------------------------------
# Stage 2 — minimal runtime image
# ---------------------------------------------------------------------------
FROM scratch

COPY --from=builder /bin/anti-bruteforce /anti-bruteforce

# gRPC and HTTP gateway ports
EXPOSE 50051 8080

ENTRYPOINT ["/anti-bruteforce"]
