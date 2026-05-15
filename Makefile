BINARY     := anti-bruteforce
CMD_PATH   := ./cmd/server
GO         := go
GOFLAGS    := -race
#LINT_VER   := latest # appears to be v2 now
#LINT_VER   := v2.12.2


.PHONY: all build run test lint generate clean docker-build docker-run

all: lint test build

# ---------------------------------------------------------------------------
# Build
# ---------------------------------------------------------------------------
build:
	$(GO) build -o bin/$(BINARY) $(CMD_PATH)

# ---------------------------------------------------------------------------
# Run (via docker compose with containers rebuild)
# ---------------------------------------------------------------------------
run:
	docker compose up --build

# ---------------------------------------------------------------------------
# Test
# ---------------------------------------------------------------------------
test:
	$(GO) test $(GOFLAGS) -count=100 ./...

test-short:
	$(GO) test $(GOFLAGS) -count=1 ./...

test-cover:
	$(GO) test $(GOFLAGS) -count=1 -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

test-code-in-dcoker:
	docker run --rm -it  -v "${PWD}":/app -w /app  golang:1.25 \
	bash
	##go test ./internal/...

# ---------------------------------------------------------------------------
# Lint  (always fetches the latest golangci-lint version)
# ---------------------------------------------------------------------------
lint:
	# @which golangci-lint > /dev/null 2>&1 || \
	#	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh \
	#	| sh -s -- -b $$(go env GOPATH)/bin $(LINT_VER)
	golangci-lint run ./...

# ---------------------------------------------------------------------------
# Code generation (proto → Go via buf)
# ---------------------------------------------------------------------------
generate:
	@which buf > /dev/null 2>&1 || go install github.com/bufbuild/buf/cmd/buf@latest
	buf generate
	$(GO) generate ./...

# ---------------------------------------------------------------------------
# Tidy
# ---------------------------------------------------------------------------
tidy:
	$(GO) mod tidy

# ---------------------------------------------------------------------------
# Clean
# ---------------------------------------------------------------------------
clean:
	rm -rf bin/ coverage.out coverage.html

# ---------------------------------------------------------------------------
# Docker
# ---------------------------------------------------------------------------
docker-build:
	docker build -t $(BINARY):latest .

docker-run:
	docker run --rm -p 8081:8080 -p 50052:50051 $(BINARY):latest
