.PHONY: run lint test migrate generate

run:
	go run ./cmd/gophermart

lint:
	golangci-lint run ./...

test:
	go test ./...

migrate:
	migrate -path migrations -database $$DATABASE_URI -verbose up

generate:
	swag init -g cmd/gophermart/main.go -o docs
