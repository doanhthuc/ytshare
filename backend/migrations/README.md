# Database migrations

This project uses [`golang-migrate`](https://github.com/golang-migrate/migrate) — the Go-native equivalent of Flyway. Migrations are versioned, immutable SQL files with paired up/down scripts.

## File naming

```
<sequence>_<name>.up.sql
<sequence>_<name>.down.sql
```

`<sequence>` is a zero-padded six-digit number (`000001`, `000002`, …). Never edit a file that has already been applied; create a new one.

## Commands

From `backend/`:

```bash
make migrate-up                    # apply all pending migrations
make migrate-down                  # roll back the most recent migration
make migrate-status                # current schema version
make migrate-create name=add_votes # scaffold the next pair of files
make migrate-force version=1       # mark a version applied (recovery)
```

These wrap `go run ./cmd/migrate` against `MIGRATIONS_DIR` (defaults to `./migrations`).

## Server startup

`cmd/server` does **not** run migrations. Migrations are an explicit operator step:

- Locally: `make migrate` after `make up` (or whenever you add a migration).
- Compose: the one-shot `migrate` service runs `up` before the `backend` service starts.
- Production: invoke `bin/migrate up` (or `go run ./cmd/migrate up`) from your deploy pipeline before rolling out the API.
