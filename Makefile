# Flatmate Matching Platform — run everything from here.
# Uses sh tooling (rm); on Windows run make from Git Bash.
DB ?= flatmate.db
ADDR ?= :8080

.PHONY: help tools sqlc tidy vet test build runbin seed fresh clean backup promote demote serve client-install client-lint client-build pwa-assets client-setup

help: ## Show targets
	@echo "serve client-setup test vet build fresh seed backup promote demote sqlc tools tidy clean client-install client-lint client-build pwa-assets"

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
	go build -o ./bin/admin ./cmd/admin
	go build -o ./bin/backup ./cmd/backup

runbin: ## Run the built server binary (used by serve)
	./bin/flatmate

serve: tidy build seed runbin ## Tidy, build, seed, and run the server binary

seed: ## Import seed profiles (idempotent)
	go run ./cmd/seed -db $(DB)

fresh: ## Reset DB and reseed (logins adaeze/tunde@example.com, password Seedpass123!)
	rm -f $(DB) $(DB)-journal
	go run ./cmd/seed -db $(DB)

backup: ## Snapshot the database into backups/
	go run ./cmd/backup -db $(DB)

promote: ## Grant admin rights (EMAIL=x@y.z)
	go run ./cmd/admin -db $(DB) -promote $(EMAIL)

demote: ## Revoke admin rights (EMAIL=x@y.z)
	go run ./cmd/admin -db $(DB) -demote $(EMAIL)

clean: ## Remove binaries and local DB
	rm -f flatmate seed flatmate.exe seed.exe $(DB) $(DB)-journal

client-setup: client-install pwa-assets client-build ## Install deps, generate PWA assets, build SPA

client-install: ## Install client dependencies (needs Node 22+)
	cd client; npm install

client-lint: ## Typecheck and lint the client
	cd client; npm run lint

client-build: ## Lint and build the SPA into client/dist
	cd client; npm run build

pwa-assets: ## Generate PWA icons from client/public/icon.svg
	cd client; npm run pwa-assets
