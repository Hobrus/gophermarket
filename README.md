# Gophermart Loyalty Service

This repository contains a minimal implementation of the "Gophermart" loyalty service. The service exposes an HTTP API with a simple health check endpoint.

## Structure

- `cmd/gophermart` – application entry point.
- `internal` – application packages.
- `pkg` – reusable utilities.
- `migrations` – SQL migrations.

## Quick start

```bash
make run
```

The server listens on port `8080` and exposes health check endpoints:
`GET /health/live` always returns `200 OK`, and `GET /health/ready` returns
`200 OK` when the database is reachable.

## Jaeger

To collect traces locally, run Jaeger:

```bash
docker compose run -d --name jaeger -p 16686:16686 -p 4318:4318 jaegertracing/all-in-one:1.57
```

Set `OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318` before starting the server.

