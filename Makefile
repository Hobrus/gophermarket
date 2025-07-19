.PHONY: run lint test

run:
	go run ./cmd/gophermart

lint:
	golangci-lint run ./...

test:
	go test ./...
