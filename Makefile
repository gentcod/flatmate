# Flatmate Matching Platform — run everything from here.
# Uses sh tooling (rm); on Windows run make from Git Bash.
DB ?= flatmate.db
ADDR ?= :8080
UI ?= spa

.PHONY: help tools sqlc tidy vet test build run brun seed fresh demo clean client-install client-lint client-build

help: ## Show targets
	@echo "tools sqlc tidy vet test build run seed fresh demo clean client-install client-lint client-build"

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
	go build -o ./bin/flatmate ./cmd/server
	go build -o ./bin/seed ./cmd/seed

run: ## Run the server (migrates on startup)
	go run ./cmd/server -addr $(ADDR) -db $(DB)

runspa: ## Run the server (migrates on startup)
	go run ./cmd/server -addr $(ADDR) -db $(DB) -ui ${UI}

runbin:
	./bin/flatmate

brun: build runbin

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

client-install: ## Install client dependencies (needs Node 22+)
	cd client && npm install

client-lint: ## Typecheck and lint the client
	cd client && npm run lint

client-build: ## Lint and build the SPA into client/dist
	cd client && npm run build
