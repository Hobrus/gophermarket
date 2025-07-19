.PHONY: run lint test migrate

run:
	go run ./cmd/gophermart

lint:
	golangci-lint run ./...

test:
	go test ./...

migrate:
	migrate -path migrations -database $$DATABASE_URI -verbose up
