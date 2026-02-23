# Makefile
.PHONY: build run clean dev

build:
	@echo "Building binary..."
	@go build -o bin/artifacts-svc cmd/main.go

run: build
	@echo "Running..."
	@./bin/artifacts-svc

clean:
	@rm -rf bin/

dev:
	@echo "Starting dev server..."
	@/home/sirkartik/go/bin/air
