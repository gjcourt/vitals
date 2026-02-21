# Makefile for biometrics

APP_NAME := biometrics
CMD_PATH := ./cmd/$(APP_NAME)
BUILD_DIR := .

.PHONY: all
all: clean lint test build

# Build the application
.PHONY: build
build:
	go build -o $(BUILD_DIR)/$(APP_NAME) $(CMD_PATH)

# Run the application
.PHONY: run
run: build
	$(BUILD_DIR)/$(APP_NAME)

# Run tests
.PHONY: test
test:
	go test -race -v ./...

# Clean build artifacts
.PHONY: clean
clean:
	rm -f $(APP_NAME)
	rm -rf bin/
	rm -f $(APP_NAME)-linux-amd64 $(APP_NAME)-darwin-amd64 $(APP_NAME)-darwin-arm64 $(APP_NAME)-windows-amd64.exe

# Development mode with hot reload (requires air)
.PHONY: dev
dev:
	@which air > /dev/null || (echo "Installing air..." && go install github.com/cosmtrek/air@latest)
	air

# Download dependencies
.PHONY: deps
deps:
	go mod download
	go mod tidy

.PHONY: install-test-deps
install-test-deps:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

.PHONY: lint
lint:
	golangci-lint run ./...

# Container image variables
REGISTRY ?= ghcr.io
REGISTRY_USER ?= gjcourt
IMAGE_TAG ?= $(shell date +%Y-%m-%d)
PLATFORM ?= linux/amd64,linux/arm64

.PHONY: image
image:
	REGISTRY=$(REGISTRY) REGISTRY_USER=$(REGISTRY_USER) IMAGE_TAG=$(IMAGE_TAG) PLATFORM=$(PLATFORM) ./scripts/build_and_push_image.sh

.PHONY: list-images
list-images:
	@echo "Fetching images for $(REGISTRY)/$(REGISTRY_USER)/$(APP_NAME)..."
	@gh api \
		-H "Accept: application/vnd.github+json" \
		-H "X-GitHub-Api-Version: 2022-11-28" \
		/users/$(REGISTRY_USER)/packages/container/$(APP_NAME)/versions \
		--jq '.[].metadata.container.tags[]' | sort -r

# Build for multiple platforms
.PHONY: build-all
build-all:
	GOOS=linux GOARCH=amd64 go build -o $(APP_NAME)-linux-amd64 $(CMD_PATH)
	GOOS=darwin GOARCH=amd64 go build -o $(APP_NAME)-darwin-amd64 $(CMD_PATH)
	GOOS=darwin GOARCH=arm64 go build -o $(APP_NAME)-darwin-arm64 $(CMD_PATH)
	GOOS=windows GOARCH=amd64 go build -o $(APP_NAME)-windows-amd64.exe $(CMD_PATH)
