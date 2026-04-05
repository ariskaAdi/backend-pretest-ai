# =============================================
# Backend Pretest AI — Makefile
# =============================================

.PHONY: run run-genkit build tidy migrate test test-cover lint help

# Default target
.DEFAULT_GOAL := help

# ── Dev ──────────────────────────────────────

run: ## Jalankan backend server (Fiber)
	go run cmd/server/main.go

run-genkit: ## Jalankan Genkit AI service
	cd genkit && go run main.go

build: ## Build binary backend ke ./bin/server
	@mkdir -p bin
	go build -o bin/server cmd/server/main.go
	@echo "✅ Build selesai: ./bin/server"

tidy: ## go mod tidy untuk backend dan genkit
	go mod tidy
	cd genkit && go mod tidy

# ── Database ─────────────────────────────────

migrate: ## Jalankan semua migration SQL ke DB
	@echo "Running migrations..."
	docker exec -i pretest-ai-postgres psql -U postgres -d pretestai < migrations/001_create_users.sql
	docker exec -i pretest-ai-postgres psql -U postgres -d pretestai < migrations/002_create_modules.sql
	docker exec -i pretest-ai-postgres psql -U postgres -d pretestai < migrations/004_create_quizzes.sql
	docker exec -i pretest-ai-postgres psql -U postgres -d pretestai < migrations/005_add_summarize_failed_to_modules.sql
	docker exec -i pretest-ai-postgres psql -U postgres -d pretestai < migrations/006_update_user_quotas.sql
	docker exec -i pretest-ai-postgres psql -U postgres -d pretestai < migrations/007_create_lynk_transactions.sql
	docker exec -i pretest-ai-postgres psql -U postgres -d pretestai < migrations/008_add_cancelled_quiz_status.sql
	docker exec -i pretest-ai-postgres psql -U postgres -d pretestai < migrations/009_create_reviews.sql
	docker exec -i pretest-ai-postgres psql -U postgres -d pretestai < migrations/010_add_user_id_to_reviews.sql
	@echo "✅ Migration selesai"

migrate-fresh: ## Drop semua tabel lalu migrate ulang (HATI-HATI: hapus semua data)
	@echo "⚠️  Dropping all tables..."
	docker exec -i pretest-ai-postgres psql -U postgres -d pretestai -c \
		"DROP TABLE IF EXISTS questions, quizzes, modules, users CASCADE; DROP TYPE IF EXISTS user_role, quiz_status CASCADE;"
	@$(MAKE) migrate

db-shell: ## Masuk ke psql shell
	docker exec -it pretest-ai-postgres psql -U postgres -d pretestai

# ── Test ─────────────────────────────────────

test: ## Jalankan semua unit test
	go test ./... -v -race

test-cover: ## Jalankan test dengan laporan coverage
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report: coverage.html"

test-service: ## Test hanya layer service
	go test ./internal/service/... -v -race

test-handler: ## Test hanya layer handler
	go test ./internal/handler/... -v

# ── Utilities ────────────────────────────────

env: ## Salin .env.example ke .env (skip kalau sudah ada)
	@if [ ! -f .env ]; then cp .env.example .env && echo "✅ .env dibuat dari .env.example"; \
	else echo "⚠️  .env sudah ada, skip"; fi

help: ## Tampilkan daftar command
	@echo ""
	@echo "Backend Pretest AI — Available Commands:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36mmake %-16s\033[0m %s\n", $$1, $$2}'
	@echo ""
