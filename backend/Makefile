.PHONY: migrate-up migrate-down migrate-create migrate-status migrate-force migrate-version help

# Database path
DB_PATH ?= perema.db
MIGRATIONS_DIR = ./migrations

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

migrate-up: ## Run all pending migrations
	@echo "Running migrations on $(DB_PATH)..."
	@migrate -path $(MIGRATIONS_DIR) -database "sqlite3://$(DB_PATH)" up
	@echo "Migrations applied successfully!"

migrate-down: ## Rollback the last migration
	@echo "Rolling back last migration on $(DB_PATH)..."
	@migrate -path $(MIGRATIONS_DIR) -database "sqlite3://$(DB_PATH)" down 1
	@echo "Migration rolled back successfully!"

migrate-create: ## Create a new migration file (usage: make migrate-create NAME=create_users_table)
	@if [ -z "$(NAME)" ]; then \
		echo "Error: NAME is required. Usage: make migrate-create NAME=your_migration_name"; \
		exit 1; \
	fi
	@migrate create -ext sql -dir $(MIGRATIONS_DIR) -seq $(NAME)
	@echo "Migration files created in $(MIGRATIONS_DIR)!"

migrate-status: ## Show current migration version
	@echo "Current migration status for $(DB_PATH):"
	@migrate -path $(MIGRATIONS_DIR) -database "sqlite3://$(DB_PATH)" version

migrate-force: ## Force set migration version (usage: make migrate-force VERSION=1)
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION is required. Usage: make migrate-force VERSION=1"; \
		exit 1; \
	fi
	@echo "Forcing migration version to $(VERSION) on $(DB_PATH)..."
	@migrate -path $(MIGRATIONS_DIR) -database "sqlite3://$(DB_PATH)" force $(VERSION)
	@echo "Migration version forced to $(VERSION)!"

migrate-version: ## Show available migration files
	@echo "Available migration files:"
	@ls -1 $(MIGRATIONS_DIR)/*.sql 2>/dev/null || echo "No migration files found"

# Development helpers
dev: ## Build and run the server in development mode
	@go build -o perema . && ./perema

test: ## Run all tests
	@go test ./... -v

build: ## Build the application
	@go build -o perema .

clean: ## Clean build artifacts
	@rm -f perema
	@echo "Build artifacts cleaned!"
