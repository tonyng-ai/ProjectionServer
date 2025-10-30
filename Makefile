.PHONY: help build run test clean frontend backend all

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

all: frontend backend ## Build both frontend and backend

deps: ## Install Go dependencies
	go mod download
	go mod verify

frontend-deps: ## Install frontend dependencies
	cd frontend && npm install

frontend: ## Build frontend
	cd frontend && npm run build

backend: deps ## Build backend
	go build -o syncservice cmd/syncservice/main.go

build: all ## Build complete application

run: ## Run the application (requires built frontend)
	./syncservice -config config/sync-config.yaml

dev-backend: ## Run backend in development mode
	go run cmd/syncservice/main.go -config config/sync-config.yaml

dev-frontend: ## Run frontend in development mode
	cd frontend && npm start

test: ## Run tests
	go test -v ./...

clean: ## Clean build artifacts
	rm -f syncservice
	rm -rf frontend/build
	rm -rf frontend/node_modules

fmt: ## Format Go code
	go fmt ./...

lint: ## Run linter
	golangci-lint run

docker-build: ## Build Docker image
	docker build -t mssql-postgres-sync:latest .

.DEFAULT_GOAL := help
