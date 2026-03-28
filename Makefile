.PHONY: all setup dev build test lint clean docker

# Variables
BINARY_NAME=hearth
VERSION?=0.1.0
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT)"

all: build

# Development setup
setup:
	@echo "Setting up development environment..."
	cd backend && go mod download
	cd frontend && npm install

# Run development servers
dev:
	@echo "Starting development servers..."
	docker-compose -f docker-compose.dev.yml up -d db redis
	make -j2 dev-backend dev-frontend

dev-backend:
	cd backend && go run ./cmd/hearth

dev-frontend:
	cd frontend && npm run dev

# Build
build: build-backend build-frontend

build-backend:
	cd backend && go build $(LDFLAGS) -o ../bin/$(BINARY_NAME) ./cmd/hearth

build-frontend:
	cd frontend && npm run build

# Testing
test: test-backend test-frontend

test-backend:
	cd backend && go test -v ./...

test-frontend:
	cd frontend && npm test

test-coverage:
	cd backend && go test -coverprofile=coverage.out ./...
	cd backend && go tool cover -html=coverage.out -o coverage.html

# Linting
lint: lint-backend lint-frontend

lint-backend:
	cd backend && golangci-lint run

lint-frontend:
	cd frontend && npm run lint

# Docker
docker:
	docker build -t hearth:$(VERSION) .

docker-push:
	docker tag hearth:$(VERSION) ghcr.io/ghndrx/hearth:$(VERSION)
	docker tag hearth:$(VERSION) ghcr.io/ghndrx/hearth:latest
	docker push ghcr.io/ghndrx/hearth:$(VERSION)
	docker push ghcr.io/ghndrx/hearth:latest

# Database
migrate-up:
	cd backend && go run ./cmd/migrate up

migrate-down:
	cd backend && go run ./cmd/migrate down

migrate-new:
	@read -p "Migration name: " name; \
	cd backend && go run ./cmd/migrate new $$name

# Cleanup
clean:
	rm -rf bin/
	rm -rf backend/coverage.*
	rm -rf frontend/build/
	rm -rf frontend/.svelte-kit/

# Help
help:
	@echo "Hearth Development Commands"
	@echo ""
	@echo "  setup         Install dependencies"
	@echo "  dev           Run development servers"
	@echo "  build         Build backend and frontend"
	@echo "  test          Run all tests"
	@echo "  lint          Run linters"
	@echo "  docker        Build Docker image"
	@echo "  clean         Remove build artifacts"
