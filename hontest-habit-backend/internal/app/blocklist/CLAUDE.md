# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Scope

This covers `internal/app/blocklist/` (manager, validation) plus its `repository/` and `models/` subpackages. Creating, listing, and soft-removing a user's blocked-site entries is implemented end-to-end, including HTTP wiring: `internal/webserver/routes/blocklist_routes.go` registers `POST /blocklist/entries`, `GET /blocklist/entries`, and `DELETE /blocklist/entries/{id}`, and `cmd/main.go` constructs the `BlocklistRepository`/`BlocklistManager` and mounts the route group behind `middlewares.Authenticate` — the first protected route group in the repo. See the root CLAUDE.md's Webserver/Middlewares sections for the HTTP layer itself.

## User identity: plain `string`, not `types.UserId`

Every method here takes a `userID string` — the real `users.id` UUID, cast to text, read off the validated JWT claims (`claims.Subject`). `types.UserId` (`int64`) is declared in `internal/types/types.go` but deliberately unused here: it can't represent the actual UUID-shaped identity, and `auth`'s repository already bypasses it the same way (`CreateUser` etc. return a plain `string`). Don't introduce `types.UserId` into this package's signatures — it would silently misrepresent the value's real shape. `blocklist_entries.user_id` is `UUID NOT NULL` with no `REFERENCES users(id)` constraint (an explicit choice, not an oversight).

## Duplicate check, deliberately dual-layer

Mirrors `auth`'s email-uniqueness pattern: `BlocklistManager.CreateEntry` calls `repository.EntryExists` first (fast pre-check, clear `errors.Conflict`), and `repository.CreateEntry` *also* inserts via `INSERT ... ON CONFLICT (user_id, url) WHERE is_active DO NOTHING RETURNING ...` and treats a resulting `pgx.ErrNoRows` (checked with `goerrors.Is`) as `errors.Conflict` — the race-condition backstop for two concurrent creates of the same URL. Keep both.

The uniqueness index is partial (`WHERE is_active`), so a soft-deleted entry doesn't block re-adding the same URL later — the `ON CONFLICT` target's `WHERE is_active` clause must match the index's predicate exactly for Postgres to pick it as the arbiter.

## Soft delete

`RemoveEntry` never deletes a row — it sets `is_active = false`. `GetEntries` and `EntryExists` both filter on `is_active`. `RemoveEntry`'s `UPDATE ... WHERE id = $1 AND user_id = $2 AND is_active` scopes by both id and the acting user, so removing another user's entry (or an already-removed one) returns `errors.NotFound`, not `errors.Forbidden` — the ownership mismatch and "already gone" cases are intentionally indistinguishable to the caller, avoiding leaking whether an entry id belongs to someone else.

## URL normalization (`validate.go`)

`validateURL` stores a bare `scheme://host` — no path, query, or fragment, both lowercased. A bare `example.com` (no scheme) is treated as `https://example.com`. This means blocking a domain blocks everything under it; this package does not support path-level blocking. If that's ever needed, it's a deliberate change to `validateURL` and the uniqueness index, not a bug.

## Meta / visits

`models.Meta{ Visits []*Visit }` is stored as a single `JSONB` column (`blocklist_entries.meta`), not a normalized table — its shape is explicitly unsettled (see the `TODO` on `models.Meta`), so this avoids committing to a schema prematurely. Nothing currently writes to `Meta` after creation (it's always `{}` on insert); appending visits is unimplemented.

## `UpdateEntry` is not implemented

There's no `BlocklistManager.UpdateEntry` and no `PUT`/`PATCH` route. The original stub's signature took no parameters and its mutable-field contract was never decided — can `url`/`frequency`/`limit` change, does changing `url` need to re-run the duplicate check, is `meta` ever client-mutable at all? Don't add a no-op stub for this; it would look like a working update to any caller. Design the contract first (see the `TODO(blocklist)` comment in `manager.go`), then add both the manager method and its route together.

## Migration

`internal/platform/db/migrations/000003_create_blocklist_entries_table.{up,down}.sql` adds `blocklist_entries` (`id BIGSERIAL`, `user_id UUID` unconstrained, `url TEXT`, optional daily time window, `frequency`/`limit_count`, `meta JSONB`, `is_active`, timestamps), plus the partial unique index backing the duplicate check and a plain index on `user_id` for `GetEntries`.
