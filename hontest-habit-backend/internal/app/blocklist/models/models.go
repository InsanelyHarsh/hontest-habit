package models

import (
	"time"

	"github.com/insanelyharsh/hontest-habit/internal/types"
)

// CreateEntryRequest is the client-facing payload for adding a blocked
// site. The acting user comes from the authenticated JWT claims, never
// from this struct — see internal/webserver/routes/blocklist_routes.go.
type CreateEntryRequest struct {
	URL            string         `json:"url"`
	DailyStartTime *time.Time     `json:"daily_start_time,omitempty"`
	DailyEndTime   *time.Time     `json:"daily_end_time,omitempty"`
	Limit          FrequencyLimit `json:"limit"`
}

type FrequencyLimit struct {
	Frequency types.Frequency `json:"frequency"`
	Limit     int             `json:"limit"` // limit per freq, e.g. 4 daily, 10 monthly
}

// BlocklistEntry is a stored blocked-site entry.
type BlocklistEntry struct {
	ID             types.BlocklistId `json:"id"`
	URL            string            `json:"url"`
	DailyStartTime *time.Time        `json:"daily_start_time,omitempty"`
	DailyEndTime   *time.Time        `json:"daily_end_time,omitempty"`
	Limit          FrequencyLimit    `json:"limit"`
	Meta           Meta              `json:"meta"`
	IsActive       bool              `json:"is_active"` // false -> soft-deleted
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

// TODO: improve/structure meta further — shape intentionally minimal for now.
type Meta struct {
	Visits []*Visit `json:"visits"`
}

type Visit struct {
	Timestamp time.Time `json:"timestamp"`
}
