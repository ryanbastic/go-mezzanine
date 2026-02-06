.PHONY: build run test clean tidy

build:
	go build -o bin/mezzanine ./cmd/mezzanine

run: build
	./bin/mezzanine

test:
	go test ./...

tidy:
	go mod tidy

clean:
	rm -rf bin/

claude:
	claude --dangerously-skip-permissions

claude_resume:
	claude --resume dd93d1da-a859-4148-b51d-6dff9219e84b

up:
	docker compose up -d --build

down:
	docker compose down
