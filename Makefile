APP ?= gmg
GOFLAGS_BUILD = -trimpath -ldflags="-s -w"

.PHONY: help web build dev test check e2e image image-emulator up down logs clean

help: ## Show this help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z0-9_-]+:.*?## / {printf "\033[36m%-16s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help

web/node_modules: web/package-lock.json
	cd web && npm ci --no-audit --no-fund
	@touch $@

web: web/node_modules ## Build the web UI into web/dist (embedded in the server)
	cd web && npm run build

build: web ## Build bin/gmg (with the UI) and bin/gmg-emulator
	CGO_ENABLED=0 go build $(GOFLAGS_BUILD) -o bin/gmg ./cmd/gmg
	CGO_ENABLED=0 go build $(GOFLAGS_BUILD) -o bin/gmg-emulator ./cmd/gmg-emulator

dev: web/node_modules ## Run emulator, server (:3001) and Vite dev server (:3000) locally
	@trap 'kill 0' EXIT INT TERM; \
	go run ./cmd/gmg-emulator --address 127.0.0.1 --port 8080 & \
	GMG_GRILL_HOST=127.0.0.1 GMG_DB_PATH=data GMG_LOG_LEVEL=debug go run ./cmd/gmg & \
	cd web && npm run dev

test: web/node_modules ## Run Go and web unit tests
	go test -race ./...
	cd web && npm test

check: web/node_modules ## Formatting, vet and type checks
	@test -z "$$(gofmt -l cmd internal web)" || { gofmt -l cmd internal web; echo "run gofmt -w"; exit 1; }
	go vet ./...
	cd web && npm run check

e2e: ## Build the Docker images and run browser end-to-end checks
	./e2e/run.sh

image: ## Build the server Docker image
	docker build -t $(APP):latest .

image-emulator: ## Build the emulator Docker image
	docker build -t $(APP)-emulator:latest -f Dockerfile.emulator .

up: ## Start server + emulator with Docker Compose (http://localhost:3000)
	docker compose up --build -d

down: ## Stop Docker Compose
	docker compose down

logs: ## Tail Docker Compose logs
	docker compose logs -f

clean: ## Remove build output (bin, node_modules, e2e output, local data, built UI)
	rm -rf bin web/node_modules e2e/node_modules e2e/out data
	find web/dist -mindepth 1 ! -name .gitkeep -delete
