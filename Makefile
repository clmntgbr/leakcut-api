COMPOSE_DEV := docker compose -f compose.dev.yaml

# ============================================
# Development (compose.dev.yaml)
# ============================================

dev:
	$(COMPOSE_DEV) up -d

dev-restart:
	$(COMPOSE_DEV) down
	$(COMPOSE_DEV) up -d

build:
	$(COMPOSE_DEV) up -d --build

dev-down:
	$(COMPOSE_DEV) down

dev-logs:
	$(COMPOSE_DEV) logs -f api worker frame ocr paddleocr

api-logs:
	$(COMPOSE_DEV) logs -f api

worker-logs:
	$(COMPOSE_DEV) logs -f worker

executor-logs:
	$(COMPOSE_DEV) logs -f executor

scheduler-logs:
	$(COMPOSE_DEV) logs -f scheduler

frame-logs:
	$(COMPOSE_DEV) logs -f frame

ocr-logs:
	$(COMPOSE_DEV) logs -f ocr paddleocr

restart:
	$(COMPOSE_DEV) restart api worker frame ocr

lint:
	$(COMPOSE_DEV) exec api golangci-lint run --fix

# ============================================
# Tests (handler HTTP, via api container)
# ============================================

tests:
	$(COMPOSE_DEV) exec api go test ./internal/interfaces/http/handler/test/... -v -count=1

coverage:
	$(COMPOSE_DEV) exec api go test -coverprofile=coverage.out ./internal/interfaces/http/handler/test/... -coverpkg=./internal/interfaces/http/handler/...

coverage-html: coverage
	$(COMPOSE_DEV) exec api go tool cover -html=coverage.out -o coverage.html

# ============================================
# CLI Commands (via Docker)
# ============================================

cli-build:
	@$(COMPOSE_DEV) exec api go build -o bin/cli ./cmd/cli

migrate: cli-build
	@echo "Applying SQL migrations..."
	@$(COMPOSE_DEV) exec api ./bin/cli migrate up
	@echo "Checking model/DB schema..."
	@$(COMPOSE_DEV) exec api ./bin/cli migrate check

migrate-status: cli-build
	@$(COMPOSE_DEV) exec api ./bin/cli migrate status

migrate-down: cli-build
	@$(COMPOSE_DEV) exec api ./bin/cli migrate down

migrate-check: cli-build
	@$(COMPOSE_DEV) exec api ./bin/cli migrate check

shell:
	$(COMPOSE_DEV) exec api sh