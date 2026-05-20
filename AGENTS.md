# Repository Guidelines

## Project Structure & Module Organization

This Go module is `github.com/staticbackendhq/core`. Root `.go` files implement
the public API and server behavior, with matching `*_test.go` files beside
them. The executable entry point is in `cmd/`. Provider and feature code lives
in focused packages: `database/` (`memory`, `sqlite`, `postgresql`, `mongo`),
`cache/`, `storage/`, `email/`, `search/`, `function/`, `middleware/`, and
`realtime/`. Web UI assets are in `templates/` and `static/`. SQL files are in
`sql/` and `database/sqlite/sql/`; plugins are in `plugins/`.

## Build, Test, and Development Commands

- `make build`: builds `cmd/staticbackend` with version metadata.
- `make plugin`: builds the `topdf` Go plugin.
- `make start`: builds the binary and plugin, then starts the local server.
- `make test-local`: runs the main local regression suite using the default
  memory/SQLite setup and no external PostgreSQL, MongoDB, Redis, or S3
  services.
- `make alltest`: runs `go test --cover ./...` after clearing the test cache.
  This includes provider packages that expect external services.
- `make test-core`: runs root package tests after removing generated local
  database and search files.
- `make test-ci-local-clean`: runs the Docker-backed local CI suite and cleans
  services afterward.
- `make lint`: runs `golangci-lint run --timeout=10m`.

Use `ENV_FILE=.env.test.pg` or `ENV_FILE=.env.test.mongo` for provider-specific
checks.

## Coding Style & Naming Conventions

Use Go 1.25 as declared in `go.mod`. Format Go changes with `gofmt`; the
project uses tabs for indentation. Keep package names short and lowercase. Name
tests `TestXxx` in `_test.go` files beside the code under test. Prefer existing
interfaces and provider patterns before adding abstractions.

## Testing Guidelines

Tests use Go's standard `testing` package. Add or update tests for behavior
changes, especially shared API paths and database/cache providers. The default
`.env` is intentionally local-only (`DATABASE_URL=mem`, `DATA_STORE=memory`,
`REDIS_HOST=mem`, local storage) so agents and contributors can run bug-fix
checks without external services. Run the narrowest relevant target first, such
as `make test-search`, `make test-sqlite`, or a focused `go test ./function`;
then run `make test-local` for broader local coverage. Use `make alltest` only
when provider services are available, or `make test-ci-local-clean` to start the
Docker-backed PostgreSQL, MongoDB, Redis, and Mailpit stack.

## Commit & Pull Request Guidelines

Don't commit your changes. Maintainers and contributors always review code written by agents before commiting changes.

## Security & Configuration Tips

Do not commit real secrets or local `.env` files. Required local settings include
`APP_SECRET`, `JWT_SECRET`, `DATA_STORE`, `DATABASE_URL`, mail provider values,
cache settings, and storage provider values. Use memory or SQLite providers for
lightweight local testing when external services are unnecessary.
