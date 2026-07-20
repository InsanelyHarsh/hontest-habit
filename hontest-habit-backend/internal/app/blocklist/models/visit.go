package models

import (
	"time"

	"github.com/insanelyharsh/hontest-habit/internal/types"
)

// VisitCounter reports how much of a blocklist entry's limit has been used,
// and how much remains, within its current period (the window is derived
// from the entry's own Limit.Frequency — see blocklist.computePeriodBounds).
type VisitCounter struct {
	BlocklistID types.BlocklistId `json:"blocklist_id"`
	Frequency   types.Frequency   `json:"frequency"`
	PeriodStart time.Time         `json:"period_start"`
	PeriodEnd   time.Time         `json:"period_end"`
	Limit       int               `json:"limit"`
	UsedCount   int               `json:"used_count"`
	// Remaining is Limit - UsedCount and is NOT clamped at 0: a negative
	// value signals the entry is over its limit for the current period.
	// Recording a visit never fails because of this (see manager.RecordVisit).
	Remaining int `json:"remaining"`
}
