.PHONY: build run test clean tidy openapi client

build:
	go build -o bin/mezzanine ./cmd/mezzanine

run: build
	SHARD_CONFIG_PATH=shards.json ./bin/mezzanine

test:
	go test ./...

tidy:
	go mod tidy

clean:
	rm -rf bin/ pkg/mezzanine/ openapi.json

openapi:
	@echo "Fetching OpenAPI spec from running server..."
	curl -sf http://localhost:8080/openapi.json > openapi.json
	@echo "Wrote openapi.json"

client: openapi
	docker run --rm -u $(shell id -u):$(shell id -g) -v $(PWD):/local \
		openapitools/openapi-generator-cli:v7.12.0 generate \
		-i /local/openapi.json \
		-g go \
		-o /local/pkg/mezzanine \
		--additional-properties=packageName=mezzanine \
		--git-user-id=ryanbastic \
		--git-repo-id=go-mezzanine/pkg/mezzanine
	cd pkg/mezzanine && go mod tidy
	@echo "Generated Go client in pkg/mezzanine/"

claude:
	claude --dangerously-skip-permissions

claude_resume:
	claude --resume cf97a7f9-f702-47d6-8c3f-5174be26150e

up:
	docker compose up -d --build

down:
	docker compose down -v

psql:
	psql -h localhost -p 5432 -U postgres -d mezzanine

psql2:
	psql -h localhost -p 5433 -U postgres -d mezzanine
