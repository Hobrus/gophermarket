# Gophermart Loyalty Service

This repository contains a minimal implementation of the "Gophermart" loyalty service written in Go.

## Structure

- `cmd/gophermart` – application entry point.
- `internal` – application packages.
- `pkg` – reusable utilities.
- `migrations` – SQL migrations.

## Quick start

```bash
make run
```

The server listens on port `8080` and exposes health check endpoints: `GET /health/live` always returns `200 OK`, and `GET /health/ready` returns `200 OK` when the database is reachable.

## API

OpenAPI documentation is available at `/swagger/index.html` when the service is running. The specification can also be found in [docs/swagger.yaml](docs/swagger.yaml).

## Running with Docker Compose

Start the application together with PostgreSQL:

```bash
DATABASE_URI="postgres://postgres:postgres@db:5432/gophermart?sslmode=disable" \
ACCRUAL_SYSTEM_ADDRESS="http://accrual:8080" \
JWT_SECRET="secret" \
docker compose up --build
```

Jaeger can be started separately to collect traces:

```bash
docker compose run -d --name jaeger -p 16686:16686 -p 4318:4318 jaegertracing/all-in-one:1.57
```

Set `OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318` before starting the server.

## Environment variables

| Name | Description | Default |
|------|-------------|---------|
| `RUN_ADDRESS` | HTTP listen address | `:8080` |
| `DATABASE_URI` | PostgreSQL connection string | **required** |
| `ACCRUAL_SYSTEM_ADDRESS` | URL of the accrual service | **required** |
| `JWT_SECRET` | Secret used to sign JWT tokens | **required** |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OTLP exporter endpoint for traces and metrics | *(optional)* |

## Example requests

```bash
# register a new user and store cookie
curl -c cookie.txt -H "Content-Type: application/json" \
  -d '{"login":"alice","password":"secret"}' \
  http://localhost:8080/api/user/register

# login
curl -c cookie.txt -H "Content-Type: application/json" \
  -d '{"login":"alice","password":"secret"}' \
  http://localhost:8080/api/user/login

# upload order
curl -b cookie.txt -d "12345678903" \
  http://localhost:8080/api/user/orders

# get balance
curl -b cookie.txt http://localhost:8080/api/user/balance
```
