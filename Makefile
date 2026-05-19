
date:=$(shell date +%s)
APP_STRING=gmg

ifndef DOCKER_HUB_USERNAME
APP=$(APP_STRING)
else
APP=$(DOCKER_HUB_USERNAME)/$(APP_STRING)
endif

.PHONY: manifest build image push-image prepare clean test dev dev-deps

# Local-dev orchestration (no Docker; assumes you ran `flox activate`).
# 8080 is the real grill's port; 18080 keeps the emulator out of its way and
# avoids clashes with anything else commonly bound to 8080 on a dev machine.
EMU_PORT ?= 18080

help: ## This help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST) | sort

.DEFAULT_GOAL := help

prepare:
	@echo Validating system
	@test -z $$DOCKER_HUB_USERNAME && echo 'WARN: DOCKER_HUB_USERNAME is unset. Push functions will not work.' || echo -n .
	@echo

manifest: prepare ## Generate a manifest of what's in the box
	@echo App: $(APP) > manifest
	@echo Unix timestamp: $(date) >> manifest @echo last git commit: $(shell git log --pretty=oneline |  head -1 | awk '{print $$1}') >> manifest @echo active branch: $(shell git rev-parse --abbrev-ref HEAD) >> manifest
	@echo git status: $(shell git diff --quiet || echo 'dirty') >> manifest
	@echo git has untrakced files: >> manifest
	@git status -s  >> manifest

image: manifest ## Build Docker Image
	cd src; docker build -t $(APP):$(date) -t $(APP):latest .

image-nc: manifest ## Build Docker Image with no caching
	cd src; docker build --no-cache -t $(APP):$(date) -t $(APP):latest gmg .

push-image: prepare image ## Push Docker Image to Docker hub (You may need to auth to Docker hub)
	cd src; docker push $(APP):$(date)
	cd src; docker push $(APP):latest

run: ## Helper function to run the docker image built
	./start

test: ## Run tests for gmg-client, gmg-server, gmg-app (also builds the UI as a regression net)
	cd src/gmg-client && npx jest
	cd src/gmg-server && npx jest
	cd src/gmg-app    && npm test
	# Build the UI too — this catches Vite/toolchain regressions
	# (e.g. an accidental reintroduction of --openssl-legacy-provider would
	# also resurface here under modern Node).
	cd src/gmg-app    && npm run build

clean: ## Remove manifest file and purge node_modules
	rm -rf manifest ./src/gmg-app/node_modules ./src/gmg-client/node_modules ./src/gmg-server/node_modules ./src/gmg-server/public/app

dev-deps: ## Install node + dotnet deps for the local-dev stack
	@echo "▸ installing gmg-client deps";  cd src/gmg-client   && npm install --silent --no-audit --no-fund
	@echo "▸ installing gmg-server deps";  cd src/gmg-server   && npm install --silent --no-audit --no-fund
	@echo "▸ installing gmg-app deps";     cd src/gmg-app      && npm install --silent --no-audit --no-fund
	@echo "▸ restoring gmg-emulator";      cd src/gmg-emulator && dotnet restore --verbosity quiet

dev: dev-deps ## Run emulator (udp/$(EMU_PORT)) + gmg-server (3001) + Vite (3000); no Docker
	@command -v dotnet >/dev/null || { echo "ERROR: 'dotnet' not on PATH — run 'flox activate' first"; exit 1; }
	@command -v node   >/dev/null || { echo "ERROR: 'node' not on PATH — run 'flox activate' first";   exit 1; }
	@echo ""
	@echo "▸ emulator: udp/$(EMU_PORT)   server: http://localhost:3001/   ui (HMR): http://localhost:3000/"
	@echo "▸ Ctrl-C to stop all three"
	@echo ""
	@trap 'kill 0' EXIT INT TERM; \
	  ( cd src/gmg-emulator && dotnet run --no-restore -- -p $(EMU_PORT) ) & \
	  ( cd src/gmg-server   && GMG_GRILL_HOST=127.0.0.1 GMG_GRILL_PORT=$(EMU_PORT) npm run start:dev ) & \
	  ( cd src/gmg-app      && npm start ) & \
	  wait
