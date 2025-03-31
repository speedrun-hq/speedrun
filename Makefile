# Go commands
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOCLEAN=$(GOCMD) clean
GOMOD=$(GOCMD) mod
GOWORK=$(GOCMD) work

# Binary names
BINARY_NAME=zetafast
API_BINARY_PATH=api/$(BINARY_NAME)

# Build flags
LDFLAGS=-ldflags "-w -s"

# Default target
.DEFAULT_GOAL := build

.PHONY: all build clean test run-api deps lint migrate dev docs check help install-all start-all

all: build

build:
	$(GOBUILD) $(LDFLAGS) -o $(API_BINARY_PATH) ./api
	cd frontend && npm run build

clean:
	$(GOCLEAN)
	rm -f $(API_BINARY_PATH)
	cd frontend && rm -rf build node_modules

test:
	$(GOTEST) -v ./...
	cd frontend && npm test

run-api:
	$(GOBUILD) $(LDFLAGS) -o $(API_BINARY_PATH) ./api
	./$(API_BINARY_PATH)

deps:
	$(GOGET) ./...
	cd frontend && npm install

lint:
	golangci-lint run
	cd frontend && npm run lint

migrate:
	$(GOBUILD) -o $(API_BINARY_PATH) ./api
	./$(API_BINARY_PATH) migrate

dev:
	$(GOCMD) run ./api/main.go

docs:
	swag init -g api/main.go

check: lint test

install-all:
	npm run install-all

start-all:
	npm run dev

help:
	@echo "Available commands:"
	@echo "  make build      - Build both API and frontend"
	@echo "  make clean      - Clean build files and dependencies"
	@echo "  make test       - Run tests for both API and frontend"
	@echo "  make run-api    - Run the API server"
	@echo "  make deps       - Download dependencies for both API and frontend"
	@echo "  make lint       - Run linters for both API and frontend"
	@echo "  make migrate    - Run database migrations"
	@echo "  make dev        - Run API in development mode"
	@echo "  make docs       - Generate API documentation"
	@echo "  make check      - Run linters and tests"
	@echo "  make install-all- Install all dependencies"
	@echo "  make start-all  - Start both API and frontend in development mode" 