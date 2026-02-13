.PHONY: install-test-deps
install-test-deps:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint

.PHONY: lint
lint:
	golangci-lint run ./...

.PHONY: image
image:
	./scripts/build_and_push_image.sh
