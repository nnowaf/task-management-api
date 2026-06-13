.PHONY: run build test test-race swag up down logs fmt vet tidy

run:            ## Run the API locally (needs a reachable Postgres)
	go run ./cmd/api

build:          ## Compile the binary into bin/api
	go build -o bin/api ./cmd/api

test:           ## Run all unit tests
	go test ./... -count=1

test-race:      ## Run the concurrency/idempotency tests with the race detector
	go test ./test/... -race -count=1 -v

swag:           ## Regenerate the Swagger docs from code annotations
	go run github.com/swaggo/swag/cmd/swag@v1.16.3 init -g cmd/api/main.go -o docs --parseInternal

up:             ## Build & start the full stack (Postgres + API) in Docker
	docker compose up --build -d

down:           ## Stop the stack and remove volumes
	docker compose down -v

logs:           ## Tail the API logs
	docker compose logs -f api

fmt:            ## Format the code
	gofmt -w internal cmd test

vet:            ## Static analysis
	go vet ./...

tidy:           ## Tidy go.mod/go.sum
	go mod tidy
