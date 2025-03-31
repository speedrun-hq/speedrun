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

# Docker commands
DOCKER_COMPOSE=docker compose

# Default target
.DEFAULT_GOAL := build

.PHONY: all build clean test run-api deps lint migrate dev docs check help install-all start-all docker-db-start docker-db-stop docker-db-logs docker-db-clean

all: build

build:
	cd api && $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) .
	cd frontend && npm run build

clean:
	cd api && $(GOCLEAN)
	rm -f api/$(BINARY_NAME)
	cd frontend && rm -rf build node_modules
	rm -rf node_modules

test:
	cd api && $(GOTEST) -v ./...
	cd frontend && npm test

run-api:
	cd api && $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) .
	cd api && ./$(BINARY_NAME)

deps:
	cd api && $(GOGET) ./...
	cd api && $(GOMOD) download
	cd api && $(GOMOD) verify
	cd frontend && npm install
	npm install concurrently

lint:
	cd api && golangci-lint run
	cd frontend && npm run lint

migrate:
	cd api && $(GOBUILD) -o $(BINARY_NAME) .

# Docker targets
docker-db-start:
	$(DOCKER_COMPOSE) up -d postgres
	@echo "Waiting for database to be ready..."
	@until docker exec zetafast_postgres pg_isready -U zetafast; do sleep 1; done
	@echo "Database is ready!"

docker-db-stop:
	$(DOCKER_COMPOSE) stop postgres

docker-db-logs:
	$(DOCKER_COMPOSE) logs -f postgres

docker-db-clean:
	$(DOCKER_COMPOSE) down -v

# Start all services with Docker database
start-all: docker-db-start
	GO_ENV=production npx concurrently "cd frontend && npm run dev" "cd api && go run main.go"

# Install all dependencies
install-all: deps

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
	@echo "  make docker-db-start - Start the Docker database"
	@echo "  make docker-db-stop - Stop the Docker database"
	@echo "  make docker-db-logs - View Docker database logs"
	@echo "  make docker-db-clean - Clean Docker database" 