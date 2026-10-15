# go-boilerplate

A Gin + GORM API starter with config, structured logging, migrations, caching, auth middleware, health checks and metrics wired up.

## Quick start

```sh
make env-setup        # creates .env from .env.example
make docker-up        # api (hot reload) + postgres + redis on :8080
```

Or run locally without any dependencies:

```sh
GIN_MODE=debug DB_ENABLED=false go run .
curl localhost:8080/health
```

API docs (Swagger UI) are served at `http://localhost:8080/docs` outside release mode.

## Layout

| Path | Purpose |
| --- | --- |
| `main.go` | Wiring, HTTP server with timeouts, graceful shutdown |
| `config/` | Env-based config (`ENV_FILE` picks the file; real env vars win) with validation |
| `routes/` | Middleware chain and route registration |
| `controllers/` | HTTP handlers (health, docs, example `/api/v1/me`) |
| `middleware/` | Request ID, logging, recovery, CORS, rate limit, body limit, JWT auth, roles, API key |
| `database/` | GORM connection (postgres/mysql), pool, health, stats |
| `migrations/` | Run-once tracked migrations (`schema_migrations` table) |
| `seeder/` | Named, idempotent seeders |
| `cache/` | `Store` interface with in-memory and Redis backends, stampede protection, hit/miss metrics |
| `models/` | `BaseModel` and your models |
| `utils/` | Logger, JSON response envelope, validation, JWT manager, cursor pagination, Prometheus metrics |
| `docs/` | Embedded `openapi.yaml` |
| `cmd/migrate/` | Migration and seeding CLI |

## Conventions

**Responses** — use the helpers in `utils/response.go` so every response shares one envelope:

```json
{ "success": false, "error": { "code": "NOT_FOUND", "message": "..." }, "request_id": "req_..." }
```

**Validation** — `utils.BindAndValidate(c, &req)` binds JSON and writes a 422 with per-field messages on failure.

**Migrations** — append to `migrations.All()`; each runs once, inside a transaction, and is recorded by name. Never rename or reorder applied migrations.

```sh
make migrate          # apply pending
make migrate-status   # list applied/pending
make seed             # run all seeders
```

Set `DB_AUTO_MIGRATE=true` to apply migrations on startup (enabled in docker-compose).

**Auth** — `middleware.AuthMiddleware(jwtManager)` accepts a Bearer token or the `token` cookie and stores the claims; read them with `middleware.GetClaims(c)`. Refresh tokens are rejected as access tokens. Restrict by role with `middleware.RoleMiddleware("admin")`.

**Caching** — inject `cache.Store` into services and use `GetOrSet`; concurrent misses for the same key share one load. Set `USE_REDIS=true` to switch backends.

## Endpoints

| Route | Notes |
| --- | --- |
| `GET /ping`, `/health`, `/livez`, `/readyz` | Health and Kubernetes probes |
| `GET /metrics`, `/metrics/prometheus` | Runtime and Prometheus metrics (non-release only) |
| `GET /internal/metrics/prometheus` | Prometheus metrics behind `X-API-Key` (all modes) |
| `GET /docs`, `/openapi.yaml` | API docs (non-release only) |
| `GET /api/v1/me` | Example authenticated route |

## Production

`GIN_MODE` defaults to `release`, which requires `JWT_SECRET` (32+ chars) and `DB_PASSWORD`. Set `TRUSTED_PROXIES` to your load balancer so client IPs (rate limiting, logs) are correct.

```sh
make docker-build     # distroless image containing /app/api and /app/migrate
```

## Make targets

Run `make help` for the full list (`build`, `dev`, `test`, `test-race`, `coverage`, `lint`, `fmt`, `check`, ...).
