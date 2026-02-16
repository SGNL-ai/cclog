BINARY    := cclog
VERSION   ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS   := -s -w -X main.version=$(VERSION)
GOFLAGS   := -race
COVER_MIN := 80

.PHONY: all build test lint fmt vet security coverage clean install

all: fmt vet lint test build  ## Full CI pipeline

build:  ## Compile binary
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/cclog/

test:  ## Run tests with race detector
	go test $(GOFLAGS) -count=1 ./...

coverage:  ## Coverage report + threshold check
	go test $(GOFLAGS) -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
	@total=$$(go tool cover -func=coverage.out | grep ^total | awk '{print $$3}' | tr -d '%'); \
	if [ $$(echo "$$total < $(COVER_MIN)" | bc) -eq 1 ]; then \
		echo "FAIL: coverage $$total% < $(COVER_MIN)%"; exit 1; fi
	@echo "OK: coverage meets $(COVER_MIN)% threshold"

lint:  ## Run golangci-lint
	golangci-lint run ./...

fmt:  ## Check formatting
	@test -z "$$(gofmt -l . | grep -v vendor)" || (echo "FAIL: unformatted files:"; gofmt -l . | grep -v vendor; exit 1)

vet:  ## Run go vet
	go vet ./...

security:  ## Run gosec + govulncheck
	gosec -quiet ./...
	govulncheck ./...

clean:  ## Remove build artifacts
	rm -f $(BINARY) coverage.out

install:  ## Install binary
	go install -ldflags "$(LDFLAGS)" ./cmd/cclog/
