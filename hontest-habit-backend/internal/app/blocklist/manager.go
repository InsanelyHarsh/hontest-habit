package blocklist

import (
	"context"

	"github.com/insanelyharsh/hontest-habit/internal/app/blocklist/models"
	"github.com/insanelyharsh/hontest-habit/internal/app/blocklist/repository"
	"github.com/insanelyharsh/hontest-habit/internal/common/errors"
	"github.com/insanelyharsh/hontest-habit/internal/types"
)

// BlocklistManager stores the sites a user has chosen to block. userID
// throughout is a plain string — the real users.id UUID (cast to text)
// read off the validated JWT claims, not types.UserId (declared but unused
// by any feature; see root CLAUDE.md's types/constants convention).
type BlocklistManager struct {
	repository repository.BlocklistRepository
}

func NewBlocklistManager(repo repository.BlocklistRepository) *BlocklistManager {
	return &BlocklistManager{repository: repo}
}

// CreateEntry validates req, rejects a duplicate active entry for the same
// url (repository.CreateEntry has the DB-level backstop for the
// TOCTOU race), and persists the new entry.
func (m *BlocklistManager) CreateEntry(ctx context.Context, userID string, req *models.CreateEntryRequest) (*models.BlocklistEntry, error) {
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
func (m *BlocklistManager) RemoveEntry(ctx context.Context, userID string, id types.BlocklistId) error {
	return m.repository.RemoveEntry(ctx, userID, id)
}

// GetEntries lists userID's active blocklist entries.
func (m *BlocklistManager) GetEntries(ctx context.Context, userID string) ([]*models.BlocklistEntry, error) {
	return m.repository.GetEntries(ctx, userID)
}

// TODO(blocklist): an UpdateEntry method is deliberately not implemented
// yet. Its mutable-field semantics are undecided: can url/frequency/limit
// change (and would changing url need to re-run the duplicate check)? Is
// meta ever client-mutable, or only appended internally when a visit is
// recorded? Design that contract before adding the method and a route for
// it — a stub that silently returns nil here would look like a working
// no-op update to any caller.
