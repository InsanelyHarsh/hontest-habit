CREATE TABLE blocklist_entries (
    id               BIGSERIAL PRIMARY KEY,
    user_id          UUID NOT NULL,
    url              TEXT NOT NULL,
    daily_start_time TIMESTAMPTZ,
    daily_end_time   TIMESTAMPTZ,
    frequency        TEXT NOT NULL,
    limit_count      INTEGER NOT NULL,
    meta             JSONB NOT NULL DEFAULT '{}'::jsonb,
    is_active        BOOLEAN NOT NULL DEFAULT true,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Prevents duplicate active blocks of the same URL per user; the partial
-- WHERE is_active clause means a soft-deleted entry doesn't block re-adding
-- the same URL later.
CREATE UNIQUE INDEX blocklist_entries_user_id_url_active_idx
    ON blocklist_entries (user_id, url)
    WHERE is_active;

CREATE INDEX blocklist_entries_user_id_idx ON blocklist_entries (user_id);
