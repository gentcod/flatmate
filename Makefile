# Flatmate Matching Platform — run everything from here.
# Uses sh tooling (rm); on Windows run make from Git Bash.
DB ?= flatmate.db
ADDR ?= :8080

.PHONY: help tools sqlc tidy vet test build run brun seed fresh demo clean

help: ## Show targets
	@echo "tools sqlc tidy vet test build run seed fresh demo clean"

tools: ## Install codegen tools (one-time)
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

sqlc: ## Generate type-safe DB code from sql/queries
	sqlc generate

tidy: ## Resolve dependencies
	go mod tidy

vet: ## Static checks
	go vet ./...

test: ## Unit tests
	go test ./...

build: ## Build server and seeder binaries
	gofmt -l -s -w .
	go build -o /bin/flatmate ./cmd/server
	go build -o /bin/seed ./cmd/seed

run: ## Run the server (migrates on startup)
	go run ./cmd/server -addr $(ADDR) -db $(DB)

brun: build run

seed: ## Import seed profiles (idempotent)
	go run ./cmd/seed -db $(DB)

fresh: ## Reset DB and reseed
	rm -f $(DB) $(DB)-journal
	go run ./cmd/seed -db $(DB)

demo: fresh ## Reseed and run for the demo
	@echo "Demo logins (password Seedpass123!): adaeze@example.com, tunde@example.com"
	go run ./cmd/server -addr $(ADDR) -db $(DB)

clean: ## Remove binaries and local DB
	rm -f flatmate seed flatmate.exe seed.exe $(DB) $(DB)-journal
