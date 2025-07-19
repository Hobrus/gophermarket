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

The server listens on port `8080` and responds to `GET /ping` with `200 OK`.

