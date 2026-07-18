# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
go build ./...                  # build everything
go vet ./...                    # static checks
go run ./cmd                    # run the app (server mode by default)
go mod tidy                     # after adding/removing an import — always run this, not manual go.mod edits

docker compose build            # build the app image
docker compose up               # postgres + redis + migrator (one-shot) + hontest-habit-backend
```

There is no test suite yet — no `_test.go` files exist in the repo.

`docker compose config` is useful to validate `docker-compose.yml` without a running daemon.

## Architecture

### Module & layout

Module `github.com/insanelyharsh/hontest-habit`, Go 1.26.5. Standard layout:
- `cmd/main.go` — the single entrypoint/binary.
- `internal/platform/*` — infra clients (`db`, `redis`, `viper`). `viper.go` is currently an empty stub; env/`.env` bootstrapping is done inline in `cmd/main.go` instead (see below), not through a wrapper package.
- `internal/app/<feature>/` — feature modules, each split into `manager.go` (business logic / use cases), `models/` (request/response DTOs), `repository/` (persistence interface + impl). `auth` is fully implemented end-to-end, including HTTP wiring — see `internal/app/auth/CLAUDE.md` for that feature's architecture. `blocklist` (site-blocking entries for a browser extension) is a second feature module in the same shape, also fully implemented end-to-end: `BlocklistManager`/`BlocklistRepositoryImpl` support creating, listing, and soft-removing a user's blocked-site entries. `UpdateEntry` is deliberately not implemented or routed yet — its mutable-field semantics are undecided; see the `TODO(blocklist)` comment in `internal/app/blocklist/manager.go`.
- `internal/webserver/` — HTTP layer: `webserver.go` (server/router + the error-handling seam), `middlewares/` (`TraceID`, `Authenticate`), `routes/` (`auth_routes.go` registers `/auth/signup` and `/auth/login`; `blocklist_routes.go` registers `POST/GET /blocklist/entries` and `DELETE /blocklist/entries/{id}` — the first routes in the repo wrapped in `middlewares.Authenticate`). See the Webserver section below for how these fit together.
- `internal/common/errors/` — shared categorized error type.
- `internal/constants/` and `internal/types/` — see convention below.
- Top-level `dtos/` (e.g. `dtos/auth.go`) is an empty, unreferenced stub — request/response DTOs actually live in `internal/app/<feature>/models/`, following the layout above. Don't add to `dtos/` without first confirming that's an intentional change of convention.

### `internal/types` / `internal/constants` convention

**Always** put custom named types (enum-like string/int types, and cross-cutting types like a context-value key type) in `internal/types/types.go` (a flat, single-file package) and put constants of those types in `internal/constants/constants.go`, which imports `internal/types` — follow the existing pattern (e.g. `types.AppMode` + `constants.AppModeServer`/`constants.AppModeMigrator`; `types.ContextKey` + `constants.TraceIDContextKey`/`ClaimsContextKey`; `types.Frequency` + `constants.DailyFrequency`/etc.). Do not declare new named types inline in feature packages — this holds even for a type that's only conceptually owned by one feature (e.g. `types.BlocklistId` still lives in the shared `types.go`, not under `internal/app/blocklist/`). Note `types.UserId` is declared but not actually used by any feature — both `auth` and `blocklist` key on the plain `string` user ID (the real `users.id` UUID, cast to text) read off the validated JWT claims, not a named type.

Plain primitive constants that aren't a custom enum-like type (bcrypt cost, min/max lengths, TTLs, etc.) don't need an invented type — they live directly in a per-feature file under `internal/constants/` (e.g. `internal/constants/auth.go`) instead of `constants.go`. `internal/constants/constants.go` itself is reserved for constants of `internal/types` types.

### App modes: server vs. migrator

The same built binary behaves differently based on the `APP_MODE` env var, read via viper in `cmd/main.go` and switched on there (`runServer()` vs `runMigrator()`):
- `server` (default) — connects to Postgres (`internal/platform/db`) and Redis (`internal/platform/redis`), then blocks in `webserver.Server.InitWebServer()`.
- `migrator` — runs `db.RunMigrations` and exits; does not touch Redis or the webserver.

This is why `docker-compose.yml` has a one-shot `migrator` service (`APP_MODE=migrator`) that the main `hontest-habit-backend` service depends on via `condition: service_completed_successfully`, ahead of `APP_MODE=server`. Both services build from the same `DockerFile`/image — only the env var differs.

### Config

Config is read via `github.com/spf13/viper`'s **global package functions** directly (`viper.GetString(...)`, `viper.GetInt(...)`) at the point of use — there's no injected config struct passed around, and no central config type. Each package that needs config exposes its own `Config` struct + `ConfigFromEnv()` constructor that reads the relevant viper keys (see `db.ConfigFromEnv()`, `redis.ConfigFromEnv()`). `.env` loading (`SetConfigFile`, `SetConfigType`, `AutomaticEnv`, `ReadInConfig`) happens once, inline, at the top of `cmd/main.go` — a missing `.env` file only logs a warning (env vars from the process/container are still picked up via `AutomaticEnv`). Env var names and defaults are documented in `.env.sample`.

### DB (`internal/platform/db/db.go`)

- App queries go through a `pgxpool.Pool` (`db.New`), built via `github.com/jackc/pgx/v5/pgxpool`.
- Migrations use `github.com/golang-migrate/migrate/v4` with its `pgx/v5` database driver (registers the `pgx5://` scheme) and an `iofs` source backed by `//go:embed migrations/*.sql` — the migrations directory is compiled into the binary, so no separate files need to ship alongside it. `db.RunMigrations` opens its own short-lived connection via migrate; it does not share the `pgxpool.Pool`.
- Migration files live in `internal/platform/db/migrations/`, named `NNNNNN_description.{up,down}.sql`.
- Both the pool DSN (`postgres://`) and the migrate DSN (`pgx5://`) are built from the same `db.Config` via `Config.dsn(scheme)`, using `net/url` so credentials are escaped correctly — don't hand-build connection strings with `fmt.Sprintf`.
- `db.IDB` is the interface (`Query`/`QueryRow`/`Exec`/`Begin`) that repositories depend on instead of the concrete `*pgxpool.Pool`, so they're constructible against a fake in tests — see `auth/repository` and `blocklist/repository` for the pattern (`NewXRepository(conn db.IDB)`).

### Errors (`internal/common/errors/errors.go`)

`HError{ Code ErrorCode; Message string; Err error }` is the shared application error type: category constants (`CodeBadRequest`, `CodeUnauthorized`, `CodeForbidden`, `CodeNotFound`, `CodeConflict`, `CodeInternal`) each map to an HTTP status via `statusByCode`. Construct with the per-category helpers (`errors.BadRequest(msg, cause)`, `errors.NotFound(msg, cause)`, etc.) rather than `HError{}` literals or the generic `New`. `HError` implements `Unwrap()` so `errors.Is`/`errors.As` traverse into the wrapped cause. Use the package-level `errors.StatusCode(err)` (built on the generic `errors.AsType[*HError]`, Go 1.26+) to map any error chain to an HTTP status. Note the package is named `errors`, shadowing the stdlib — it imports stdlib `errors` internally as `goerrors`.

### Webserver (`internal/webserver/`)

Plain `net/http` with Go 1.22+ method-prefixed `ServeMux` patterns (`"GET "+path`) — no third-party router.

- **Construction**: use `webserver.NewServer()`, not `&webserver.Server{}` — the zero value's mux is nil, so `NewGroup` would panic. `NewServer` creates the mux immediately so routes can be registered before `InitWebServer()` is called (see `cmd/main.go`'s `runServer()`).
- **Groups**: `Server.NewGroup(prefix)` mounts a sub-`ServeMux` at `prefix` (e.g. `"/auth/"`) via `http.StripPrefix` and returns a `Group` with `GET`/`POST`/`DELETE`. The `StripPrefix` matters: `prefix` is a subtree pattern, so `net/http` dispatches the *unmodified* request path (`/auth/signup`) to the sub-mux, but the sub-mux's own patterns are registered relative to the group (`/signup`) — without stripping, every grouped route 404s. This bit the first real routes added to this repo; if `Group` ever grows a new registration method, make sure it still goes through the same stripped sub-mux. `Group.Use(mw)` returns a copy of the `Group` with one more middleware appended, applied to every route registered on that copy.
- **Controllers**: a feature's route file implements `webserver.Controller` (`Routes(grp Group)`) — a struct holding the feature's manager, a `NewXController(mgr)` constructor, and a `Routes` method that registers its handlers on the group it's given (see `routes.AuthController`/`routes.BlockListController` in `routes/auth_routes.go`/`routes/blocklist_routes.go`). `Server.Register(prefix, ctrl, mw...)` is the standard way to mount one: it composes `NewGroup(prefix)` + `Group.Use(mw)` for each middleware + `ctrl.Routes(group)` in one call — see `cmd/main.go`'s `runServer()` (`server.Register("/auth/", routes.NewAuthController(authManager))`, `server.Register("/blocklist/", routes.NewBlocklistController(blocklistManager), middlewares.Authenticate(jwtCfg))`). A new feature's route file should follow the same `XController` shape rather than reintroducing free functions.
- **Handlers return `error`**: `Group.GET`/`POST`/`DELETE` take a `webserver.HandlerFunc` (`func(w, r) error`), not a raw `http.HandlerFunc` — internally wrapped via `webserver.Wrap` (and any middleware from `Use`). A handler does its work and either writes a success response itself (via `webserver.WriteJSON`) and returns `nil`, or just `return`s an error and lets `Wrap` translate it: `errors.StatusCode(err)` picks the HTTP status, and the JSON body's `error` field is the `*errors.HError.Message` for a categorized error (never the raw wrapped cause, which may contain internal details like driver error text) or a generic `"internal server error"` string otherwise. `webserver.DecodeJSON(r, &dst)` decodes a JSON body, returning an `errors.BadRequest` on failure. See `routes/auth_routes.go` and `routes/blocklist_routes.go` for the pattern.
- **`InitWebServer()`** returns an `error` (blocks on `ListenAndServe`) rather than panicking — repo-wide convention: init functions return errors, and `cmd/main.go` is the only place that logs via `slog.Error` and calls `os.Exit(1)` on a fatal startup failure. Don't introduce `panic` for init-time errors elsewhere. It also wraps the whole mux with `middlewares.TraceID`, so every request (not just a specific group) gets a trace ID.

### Middlewares (`internal/webserver/middlewares/`)

Standard `func(http.Handler) http.Handler` middleware, composed by wrapping. `webserver.Group` has a `Use(mw)` method (returns a modified copy of the `Group`, so a base group can spin off both a protected and an unprotected sub-group) that chains middleware in front of every route registered on the returned `Group` — see the Webserver section above.

- `TraceID` — reads an incoming `X-Request-Id` header or generates one, sets it on the response and on the request context (`constants.TraceIDContextKey`, a `types.ContextKey`); read it back via `TraceIDFromContext`. Applied globally in `InitWebServer`.
- `Authenticate(cfg auth.JWTConfig)` — validates `Authorization: Bearer <token>` via `auth.ValidateToken` and stores the claims on the context (`constants.ClaimsContextKey`); read back via `ClaimsFromContext`. Takes a plain `auth.JWTConfig`, not an `*auth.AuthManager` — the middleware only needs to verify a token, not the manager's repository dependency. Attached to the `blocklist` group in `cmd/main.go` (`server.NewGroup("/blocklist/").Use(middlewares.Authenticate(jwtCfg))`) — the first protected routes in the repo.
- Deliberately, `middlewares` does not import `webserver` (it writes its own JSON error bodies inline) even though `webserver` imports `middlewares` for the global `TraceID` wrap — importing back would cycle.

### Logging

`log/slog` everywhere (JSON handler set up once in `cmd/main.go`), never `fmt.Print*`/`log.Print*` for diagnostic output.
