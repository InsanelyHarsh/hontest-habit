CREATE TABLE blocklist_visits (
    id           BIGSERIAL PRIMARY KEY,
    blocklist_id BIGINT NOT NULL,
    user_id      BIGINT NOT NULL,
    url          TEXT NOT NULL,
    visited_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- No FK to blocklist_entries(id), consistent with blocklist_entries.user_id
-- deliberately having no FK to users(id). url is denormalized from the
-- entry at visit time so history is preserved even if the entry is later
-- soft-deleted.
CREATE INDEX blocklist_visits_blocklist_id_visited_at_idx
    ON blocklist_visits (blocklist_id, visited_at);
