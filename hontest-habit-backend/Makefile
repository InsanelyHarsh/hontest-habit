.DEFAULT_GOAL := help

.PHONY: build vet run tidy migrate docker-build up up-d down logs config postgres_connect help

build: ## Build everything
	go build ./...

vet: ## Run static checks
	go vet ./...

run: ## Run the app (server mode by default)
	go run ./cmd

tidy: ## Tidy go.mod/go.sum after adding/removing an import
	go mod tidy

migrate: ## Run pending DB migrations locally (against whatever .env/env points at)
	APP_MODE=migrator go run ./cmd

docker-build: ## Build the app image
	docker compose build

up: ## Start postgres + redis + migrator (one-shot) + hontest-habit-backend
	docker compose up

up-d: ## Same as up, but detached
	docker compose up -d

down: ## Stop and remove containers
	docker compose down

logs: ## Follow logs from all services
	docker compose logs -f

config: ## Validate docker-compose.yml without a running daemon
	docker compose config

postgres_connect: ## Open an interactive psql session in the dockerized postgres
	sudo docker compose exec -it postgres psql -U postgres -d hontest_habit

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'
