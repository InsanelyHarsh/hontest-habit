# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Scope

This covers `internal/app/auth/` (manager, token issuance) plus its `repository/` and `models/` subpackages. Email/password signup and login are implemented end-to-end, including HTTP wiring: `internal/webserver/routes/auth_routes.go` registers `POST /auth/signup` and `POST /auth/login`, and `cmd/main.go` constructs the `AuthRepository`/`AuthManager` and mounts the route group. See the root CLAUDE.md's Webserver/Middlewares sections for how the HTTP layer itself works (the JSON/error seam, `Group`, `TraceID`, `Authenticate`).

## Password handling: the repository owns all crypto

`AuthRepositoryImpl` (`repository/repository.go`) is the *only* place bcrypt is used. No method returns a password hash to a caller — callers pass in a plaintext password and only ever get back a user ID (or nothing):

- `CreateUser(ctx, email, plaintextPassword)` — hashes with `golang.org/x/crypto/bcrypt` at `constants.BcryptCost`, then `INSERT ... RETURNING id`.
- `ValidateCredentials(ctx, email, plaintextPassword)` — fetches the stored hash internally and compares with `bcrypt.CompareHashAndPassword`; the hash itself never leaves this method.
- `EmailExists(ctx, email)` — `SELECT EXISTS(...)` existence check, used by `AuthManager.Signup` for an explicit pre-check.

Don't add a repository method that returns `password_hash` to a caller, and don't move bcrypt calls into `manager.go` — that would break the "hash never leaves the repository" invariant this package is built around.

**Duplicate email is checked twice, deliberately**: `AuthManager.Signup` calls `EmailExists` first (fast, clear `errors.Conflict`), but `CreateUser` *also* inserts via `INSERT ... ON CONFLICT (email) DO NOTHING RETURNING id` and treats a resulting `pgx.ErrNoRows` (checked with `goerrors.Is`, the same pattern `ValidateCredentials` uses) as `errors.Conflict` — a race-condition fallback for two concurrent signups on the same email. Keep both — removing the `CreateUser`-side check would reintroduce a TOCTOU race.

**Timing-safe login**: when `ValidateCredentials` finds no matching row, it still runs a bcrypt compare against a fixed `dummyBcryptHash` before returning `errors.Unauthorized`, so an unknown email and a wrong password take comparable time (basic defense against user-enumeration via timing). Both the "no such email" and "wrong password" cases return the exact same generic `errors.Unauthorized("invalid email or password", ...)` — never a more specific error.

## Email/password validation (`validate.go`)

`validateEmail`/`validatePassword` are unexported and called from `AuthManager.Signup`/`Login`, not from the repository. `validateEmail` uses stdlib `net/mail.ParseAddress` plus an exact-match check against the trimmed input (rejects `"Display Name <a@b.com>"` forms that `ParseAddress` alone would accept), and lowercases the result. **Normalization happens once, in the manager** — the repository always receives/stores already-lowercased email, so signup and login look up the same key regardless of input casing. `validatePassword` only enforces length bounds (`constants.MinPasswordLength`/`MaxPasswordLength` — the max exists specifically to reject inputs beyond bcrypt's silent 72-byte truncation point). `AuthManager.Login` deliberately does *not* run `validatePassword`'s length check — it only checks non-empty — so a too-short password at login gets the same generic `Unauthorized` as any other wrong password, rather than a `BadRequest` that would leak password policy to an unauthenticated caller.

## JWT (`token.go`)

`JWTConfig{Secret, TTL}` + `JWTConfigFromEnv()` follow the repo-wide per-package `Config`/`ConfigFromEnv()` convention, reading `JWT_SECRET` via viper (documented in `.env.sample`); `TTL` comes from `constants.JWTExpiry`, a compile-time constant — there's no `JWT_TTL`-style env var. `cmd/main.go` fails fast (`os.Exit(1)`) if `JWT_SECRET` is empty, the same way a db/redis connection failure does. `Claims{Email string; jwt.RegisteredClaims}` — `sub` is the user ID, via `golang-jwt/jwt/v5`, HS256. `generateJwtToken` (unexported) is called by both `AuthManager.Signup` and `Login` (a successful signup auto-authenticates, same as login — both return `*models.AuthResponse{UserID, Token}`).

`ValidateToken(ctx, cfg JWTConfig, tokenString) (*Claims, error)` is exported and package-level (not a method on `*AuthManager`) specifically so `internal/webserver/middlewares.Authenticate` can depend on just a `JWTConfig`, not a full `*AuthManager` (which would drag in the repository dependency it doesn't need). Don't reintroduce an `AuthManager.ValidateToken` wrapper — call `auth.ValidateToken` directly from anything that only needs to verify a token.

`generateJwtToken` takes `userID types.UserId` and sets `Subject: strconv.FormatInt(int64(userID), 10)` — a JWT `sub` claim must be a string per spec, so `types.UserId` (the real `users.id`, `BIGSERIAL`/`int64`) is formatted once here. Any consumer of `Claims.Subject` (e.g. `blocklist_routes.go`) parses it back with `strconv.ParseInt` rather than threading a raw string through business logic.

## `internal/constants/auth.go`

Plain untyped constants (`BcryptCost`, `MinPasswordLength`, `MaxPasswordLength`, `JWTExpiry`) — not the `types.go`-backed enum pattern used for e.g. `AppMode`, since none of these are an enum-like named type (see root CLAUDE.md's types/constants convention).

## Migration

`internal/platform/db/migrations/000001_create_users_table.{up,down}.sql` adds the `users` table (`id BIGSERIAL PRIMARY KEY`, `email TEXT UNIQUE`, `password_hash TEXT`, timestamps). There's no `updated_at` refresh trigger — no update endpoint exists yet to need one.
