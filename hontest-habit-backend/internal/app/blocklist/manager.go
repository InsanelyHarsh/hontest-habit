package blocklist

import (
	"context"
	"time"

	"github.com/insanelyharsh/hontest-habit/internal/app/blocklist/models"
	"github.com/insanelyharsh/hontest-habit/internal/app/blocklist/repository"
	"github.com/insanelyharsh/hontest-habit/internal/common/errors"
	"github.com/insanelyharsh/hontest-habit/internal/types"
)

// BlocklistManager stores the sites a user has chosen to block. userID
// throughout is types.UserId — the real users.id (BIGSERIAL) — parsed from
// the JWT sub claim at the HTTP boundary; see blocklist_routes.go.
type BlocklistManager struct {
	repository repository.BlocklistRepository
}

func NewBlocklistManager(repo repository.BlocklistRepository) *BlocklistManager {
	return &BlocklistManager{repository: repo}
}

// CreateEntry validates req, rejects a duplicate active entry for the same
// url (repository.CreateEntry has the DB-level backstop for the
// TOCTOU race), and persists the new entry.
func (m *BlocklistManager) CreateEntry(ctx context.Context, userID types.UserId, req *models.CreateEntryRequest) (*models.BlocklistEntry, error) {
	url, err := validateURL(req.URL)
	if err != nil {
		return nil, err
	}
	if err := validateFrequencyLimit(req.Limit); err != nil {
		return nil, err
	}
	if err := validateTimeWindow(req.DailyStartTime, req.DailyEndTime); err != nil {
		return nil, err
	}
	req.URL = url

	exists, err := m.repository.EntryExists(ctx, userID, url)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.Conflict("url already blocked", nil)
	}

	return m.repository.CreateEntry(ctx, userID, *req)
}

// RemoveEntry soft-deletes id, scoped to userID so one user can never
// remove another user's entry.
func (m *BlocklistManager) RemoveEntry(ctx context.Context, userID types.UserId, id types.BlocklistId) error {
	return m.repository.RemoveEntry(ctx, userID, id)
}

// GetEntries lists userID's active blocklist entries.
func (m *BlocklistManager) GetEntries(ctx context.Context, userID types.UserId) ([]*models.BlocklistEntry, error) {
	return m.repository.GetEntries(ctx, userID)
}

// RecordVisit records that userID visited the site tracked by blocklist
// entry id "now" (server time, UTC), then returns the entry's updated
// usage/remaining counts for its current period. It never fails because
// the entry is over its limit — over-limit is reported via a non-positive
// Remaining, not an error.
func (m *BlocklistManager) RecordVisit(ctx context.Context, userID types.UserId, id types.BlocklistId) (*models.VisitCounter, error) {
	now := time.Now().UTC()

	entry, err := m.repository.GetActiveEntry(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if err := m.repository.RecordVisit(ctx, userID, id, entry.URL, now); err != nil {
		return nil, err
	}
	return m.counterFor(ctx, entry, now)
}

// GetRemaining reports how much of entry id's limit remains in its
// current period, without recording a visit.
func (m *BlocklistManager) GetRemaining(ctx context.Context, userID types.UserId, id types.BlocklistId) (*models.VisitCounter, error) {
	now := time.Now().UTC()

	entry, err := m.repository.GetActiveEntry(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	return m.counterFor(ctx, entry, now)
}

// counterFor computes entry's current period bounds from its own
// Limit.Frequency, counts its visits in that window, and assembles the
// response shared by RecordVisit and GetRemaining.
func (m *BlocklistManager) counterFor(ctx context.Context, entry *models.BlocklistEntry, now time.Time) (*models.VisitCounter, error) {
	start, end, err := computePeriodBounds(entry.Limit.Frequency, now)
	if err != nil {
		return nil, err
	}
	used, err := m.repository.CountVisits(ctx, entry.ID, start, end)
	if err != nil {
		return nil, err
	}
	return &models.VisitCounter{
		BlocklistID: entry.ID,
		Frequency:   entry.Limit.Frequency,
		PeriodStart: start,
		PeriodEnd:   end,
		Limit:       entry.Limit.Limit,
		UsedCount:   used,
		Remaining:   entry.Limit.Limit - used,
	}, nil
}

// TODO(blocklist): an UpdateEntry method is deliberately not implemented
// yet. Its mutable-field semantics are undecided: can url/frequency/limit
// change (and would changing url need to re-run the duplicate check)? Is
// meta ever client-mutable, or only appended internally when a visit is
// recorded? Design that contract before adding the method and a route for
// it — a stub that silently returns nil here would look like a working
// no-op update to any caller.
