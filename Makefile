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
