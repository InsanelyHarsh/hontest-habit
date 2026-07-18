package repository

import (
	"context"
	"encoding/json"
	goerrors "errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/insanelyharsh/hontest-habit/internal/app/blocklist/models"
	"github.com/insanelyharsh/hontest-habit/internal/common/errors"
	"github.com/insanelyharsh/hontest-habit/internal/platform/db"
	"github.com/insanelyharsh/hontest-habit/internal/types"
)

// BlocklistRepository persists a user's blocked-site entries. userID is
// types.UserId throughout — the real users.id (BIGSERIAL).
type BlocklistRepository interface {
	// EntryExists reports whether userID already has an active entry for
	// url. Used as a fast pre-check by the manager; CreateEntry's
	// ON CONFLICT DO NOTHING is the race-condition backstop.
	EntryExists(ctx context.Context, userID types.UserId, url string) (bool, error)

	// CreateEntry inserts a new entry and returns the full stored row.
	// Returns errors.Conflict if the insert conflicts with the partial
	// unique index on (user_id, url) WHERE is_active.
	CreateEntry(ctx context.Context, userID types.UserId, req models.CreateEntryRequest) (*models.BlocklistEntry, error)

	// GetEntries returns userID's active entries, newest first.
	GetEntries(ctx context.Context, userID types.UserId) ([]*models.BlocklistEntry, error)

	// RemoveEntry soft-deletes (is_active = false) the entry identified by
	// id, scoped to userID. Returns errors.NotFound if no active row
	// matched both id and userID — this covers a wrong id, an id
	// belonging to a different user, and an already-removed entry
	// indistinguishably, so ownership isn't leaked to the caller.
	RemoveEntry(ctx context.Context, userID types.UserId, id types.BlocklistId) error

	// GetActiveEntry returns userID's single active entry with id id.
	// Returns errors.NotFound if no active row matched both id and userID —
	// same ownership-ambiguity semantics as RemoveEntry.
	GetActiveEntry(ctx context.Context, userID types.UserId, id types.BlocklistId) (*models.BlocklistEntry, error)

	// RecordVisit inserts a visit row for blocklist entry id at visitedAt,
	// denormalizing userID and url onto the row so history survives even
	// if the entry is later soft-deleted.
	RecordVisit(ctx context.Context, userID types.UserId, id types.BlocklistId, url string, visitedAt time.Time) error

	// CountVisits returns the number of visit rows recorded for blocklist
	// entry id within [periodStart, periodEnd).
	CountVisits(ctx context.Context, id types.BlocklistId, periodStart, periodEnd time.Time) (int, error)
}

type BlocklistRepositoryImpl struct {
	db db.IDB
}

func NewBlocklistRepository(conn db.IDB) BlocklistRepository {
	return BlocklistRepositoryImpl{db: conn}
}

func (r BlocklistRepositoryImpl) EntryExists(ctx context.Context, userID types.UserId, url string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM blocklist_entries WHERE user_id = $1 AND url = $2 AND is_active)`,
		int64(userID), url,
	).Scan(&exists)
	if err != nil {
		return false, errors.Internal("failed to check entry existence", err)
	}
	return exists, nil
}

func (r BlocklistRepositoryImpl) CreateEntry(ctx context.Context, userID types.UserId, req models.CreateEntryRequest) (*models.BlocklistEntry, error) {
	var entry models.BlocklistEntry
	var frequency string
	var metaBytes []byte
	err := r.db.QueryRow(ctx, `
		INSERT INTO blocklist_entries (user_id, url, daily_start_time, daily_end_time, frequency, limit_count)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id, url) WHERE is_active DO NOTHING
		RETURNING id, url, daily_start_time, daily_end_time, frequency, limit_count, meta, is_active, created_at, updated_at
	`, int64(userID), req.URL, req.DailyStartTime, req.DailyEndTime, string(req.Limit.Frequency), req.Limit.Limit,
	).Scan(
		&entry.ID, &entry.URL, &entry.DailyStartTime, &entry.DailyEndTime,
		&frequency, &entry.Limit.Limit, &metaBytes, &entry.IsActive,
		&entry.CreatedAt, &entry.UpdatedAt,
	)
	switch {
	case err == nil:
		entry.Limit.Frequency = types.Frequency(frequency)
		if err := json.Unmarshal(metaBytes, &entry.Meta); err != nil {
			return nil, errors.Internal("failed to decode entry meta", err)
		}
		return &entry, nil
	case goerrors.Is(err, pgx.ErrNoRows):
		return nil, errors.Conflict("url already blocked", nil)
	default:
		return nil, errors.Internal("failed to create blocklist entry", err)
	}
}

func (r BlocklistRepositoryImpl) GetEntries(ctx context.Context, userID types.UserId) ([]*models.BlocklistEntry, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, url, daily_start_time, daily_end_time, frequency, limit_count, meta, is_active, created_at, updated_at
		FROM blocklist_entries
		WHERE user_id = $1 AND is_active
		ORDER BY created_at DESC
	`, int64(userID))
	if err != nil {
		return nil, errors.Internal("failed to fetch blocklist entries", err)
	}
	defer rows.Close()

	var entries []*models.BlocklistEntry
	for rows.Next() {
		var entry models.BlocklistEntry
		var frequency string
		var metaBytes []byte
		if err := rows.Scan(
			&entry.ID, &entry.URL, &entry.DailyStartTime, &entry.DailyEndTime,
			&frequency, &entry.Limit.Limit, &metaBytes, &entry.IsActive,
			&entry.CreatedAt, &entry.UpdatedAt,
		); err != nil {
			return nil, errors.Internal("failed to scan blocklist entry", err)
		}
		entry.Limit.Frequency = types.Frequency(frequency)
		if err := json.Unmarshal(metaBytes, &entry.Meta); err != nil {
			return nil, errors.Internal("failed to decode entry meta", err)
		}
		entries = append(entries, &entry)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Internal("failed to fetch blocklist entries", err)
	}
	return entries, nil
}

func (r BlocklistRepositoryImpl) RemoveEntry(ctx context.Context, userID types.UserId, id types.BlocklistId) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE blocklist_entries SET is_active = false, updated_at = now() WHERE id = $1 AND user_id = $2 AND is_active`,
		int64(id), int64(userID),
	)
	if err != nil {
		return errors.Internal("failed to remove entry", err)
	}
	if tag.RowsAffected() == 0 {
		return errors.NotFound("entry not found", nil)
	}
	return nil
}

func (r BlocklistRepositoryImpl) GetActiveEntry(ctx context.Context, userID types.UserId, id types.BlocklistId) (*models.BlocklistEntry, error) {
	var entry models.BlocklistEntry
	var frequency string
	var metaBytes []byte
	err := r.db.QueryRow(ctx, `
		SELECT id, url, daily_start_time, daily_end_time, frequency, limit_count, meta, is_active, created_at, updated_at
		FROM blocklist_entries
		WHERE id = $1 AND user_id = $2 AND is_active
	`, int64(id), int64(userID),
	).Scan(
		&entry.ID, &entry.URL, &entry.DailyStartTime, &entry.DailyEndTime,
		&frequency, &entry.Limit.Limit, &metaBytes, &entry.IsActive,
		&entry.CreatedAt, &entry.UpdatedAt,
	)
	switch {
	case err == nil:
		entry.Limit.Frequency = types.Frequency(frequency)
		if err := json.Unmarshal(metaBytes, &entry.Meta); err != nil {
			return nil, errors.Internal("failed to decode entry meta", err)
		}
		return &entry, nil
	case goerrors.Is(err, pgx.ErrNoRows):
		return nil, errors.NotFound("entry not found", nil)
	default:
		return nil, errors.Internal("failed to fetch blocklist entry", err)
	}
}

func (r BlocklistRepositoryImpl) RecordVisit(ctx context.Context, userID types.UserId, id types.BlocklistId, url string, visitedAt time.Time) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO blocklist_visits (blocklist_id, user_id, url, visited_at)
		VALUES ($1, $2, $3, $4)
	`, int64(id), int64(userID), url, visitedAt)
	if err != nil {
		return errors.Internal("failed to record visit", err)
	}
	return nil
}

func (r BlocklistRepositoryImpl) CountVisits(ctx context.Context, id types.BlocklistId, periodStart, periodEnd time.Time) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM blocklist_visits
		WHERE blocklist_id = $1 AND visited_at >= $2 AND visited_at < $3
	`, int64(id), periodStart, periodEnd,
	).Scan(&count)
	if err != nil {
		return 0, errors.Internal("failed to count visits", err)
	}
	return count, nil
}
