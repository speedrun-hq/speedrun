# Binary names
BINARY_NAME=speedrun

# Build flags
# -w: Omits DWARF symbol table information from the binary, reducing its size
# -s: Omits symbol table and debug information from the binary, further reducing size
LDFLAGS=-ldflags "-w -s"

GO_BUILD=go build $(LDFLAGS)

DOCKER_COMPOSE=docker compose

help: ## List of commands
	@echo "Available commands:\n"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo "\nUsage: make <command>"

build: build-api ## Build both API and frontend
	cd frontend && npm run build

build-api: ## Build the API server
	cd api && $(GO_BUILD) -o build/$(BINARY_NAME) cmd/speedrun/main.go

clean: ## Clean build files and dependencies
	cd api && go clean
	rm -f api/$(BINARY_NAME)
	cd frontend && rm -rf build node_modules
	rm -rf node_modules

test: ## Run tests for both API and frontend
	cd api && go test -v ./...
	cd frontend && npm test

run-api: build-api ## Run the API server
	./api/build/$(BINARY_NAME)

deps: ## Download dependencies for both API and frontend
	cd api && go get ./...
	cd api && go mod download
	cd api && go mod verify
	cd frontend && npm install
	npm install concurrently

fmt: ## Format code
	@golangci-lint fmt

lint: ## Run linters for both API and frontend
	cd api && golangci-lint run
	cd frontend && npm run lint

docker-db-start: ## Start Docker database
	$(DOCKER_COMPOSE) up -d postgres
	@echo "Waiting for database to be ready..."
	@until docker exec speedrun_postgres pg_isready -U speedrun; do sleep 1; done
	@echo "Database is ready!"

docker-db-stop: ## Stop Docker database
	$(DOCKER_COMPOSE) stop postgres

docker-db-logs: ## View Docker database logs
	$(DOCKER_COMPOSE) logs -f postgres

docker-db-clean: ## Clean Docker database
	$(DOCKER_COMPOSE) down -v

start-all: docker-db-start ## Start all services with Docker database
	GO_ENV=production npx concurrently "cd frontend && npm run dev" "cd api && go run main.go"

.PHONY: help build clean test run-api deps lint fmt
.PHONY: docker-db-start docker-db-stop docker-db-logs docker-db-clean start-all