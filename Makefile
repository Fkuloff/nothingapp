.PHONY: swagger
swagger:
	swag init -g internal/app/app.go -o ./docs

.PHONY: run
run:
	go run cmd/server/main.go

.PHONY: build
build:
	go build -o messenger.exe cmd/server/main.go

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: vet
vet:
	go vet ./...

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: lint
lint:
	golangci-lint run

.PHONY: lint-fix
lint-fix:
	golangci-lint run --fix

.PHONY: test
test:
	go test -v -race ./...

.PHONY: check
check: fmt vet lint test
	@echo "All checks passed!"

.PHONY: clean
clean:
	rm -f messenger.exe
	rm -rf docs/

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  swagger   - Generate Swagger documentation"
	@echo "  run       - Run the server"
	@echo "  build     - Build the executable"
	@echo "  tidy      - Tidy go.mod"
	@echo "  vet       - Run go vet"
	@echo "  fmt       - Format code"
	@echo "  lint      - Run golangci-lint"
	@echo "  lint-fix  - Run golangci-lint with auto-fix"
	@echo "  test      - Run tests with race detector"
	@echo "  check     - Run all checks (fmt, vet, lint, test)"
	@echo "  clean     - Remove built executable and docs"
