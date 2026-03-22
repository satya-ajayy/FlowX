APP_NAME    := flowx
MAIN        := ./cmd/flowx
BUILD_DIR   := .bin
BINARY      := $(BUILD_DIR)/$(APP_NAME)
DOCKER_IMG  := $(APP_NAME)
GO_FILES    := $(shell find . -name '*.go' -not -path './vendor/*')

.PHONY: all build run clean test vet lint fmt tidy vendor docker docker-run help

all: lint test build

## ——— Build & Run ———

build:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BINARY) $(MAIN)
	@echo "Built $(BINARY)"

run: build
	$(BINARY) -c config.yml

dev:
	go run $(MAIN) -c config.yml

## ——— Code Quality ———

test:
	go test -v -race -count=1 ./...

test-cover:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

vet:
	go vet ./...

lint: vet
	gofmt -d $(GO_FILES)

fmt:
	gofmt -w $(GO_FILES)

## ——— Dependencies ———

tidy:
	go mod tidy

vendor:
	go mod vendor

## ——— Docker ———

docker:
	docker build -t $(DOCKER_IMG):latest .

docker-run: docker
	docker run --rm -p 3625:3625 $(DOCKER_IMG):latest

## ——— Cleanup ———

clean:
	rm -rf $(BUILD_DIR) coverage.out coverage.html

## ——— Help ———

help:
	@echo ""
	@echo "Usage: make <target>"
	@echo ""
	@echo "  build        Build the binary to $(BUILD_DIR)/"
	@echo "  run          Build and run with config.yml"
	@echo "  dev          Run without building (go run)"
	@echo "  test         Run tests with race detector"
	@echo "  test-cover   Run tests and generate coverage report"
	@echo "  vet          Run go vet"
	@echo "  lint         Run vet + gofmt diff"
	@echo "  fmt          Format all Go files in place"
	@echo "  tidy         Run go mod tidy"
	@echo "  vendor       Vendor dependencies"
	@echo "  docker       Build Docker image"
	@echo "  docker-run   Build and run Docker image"
	@echo "  clean        Remove build artifacts"
	@echo ""
