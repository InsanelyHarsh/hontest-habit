package blocklist

import (
	"net/url"
	"strings"
	"time"

	"github.com/insanelyharsh/hontest-habit/internal/app/blocklist/models"
	"github.com/insanelyharsh/hontest-habit/internal/common/errors"
	"github.com/insanelyharsh/hontest-habit/internal/constants"
)

// validateURL parses and normalizes raw to a bare "scheme://host" (no
// path/query/fragment, lowercased), so blocking a site blocks everything
// under that host rather than a single page.
func validateURL(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errors.BadRequest("url is required", nil)
	}
	if len(trimmed) > constants.MaxURLLength {
		return "", errors.BadRequest("url is too long", nil)
	}
	if !strings.Contains(trimmed, "://") {
		trimmed = "https://" + trimmed
	}

	u, err := url.Parse(trimmed)
	if err != nil {
		return "", errors.BadRequest("invalid url", err)
	}
	if u.Host == "" {
		return "", errors.BadRequest("invalid url", nil)
	}
	return strings.ToLower(u.Scheme) + "://" + strings.ToLower(u.Host), nil
}

// validateFrequencyLimit checks that fl.Frequency is one of the known
// enum values and fl.Limit is positive.
func validateFrequencyLimit(fl models.FrequencyLimit) error {
	switch fl.Frequency {
	case constants.DailyFrequency, constants.WeeklyFrequency, constants.MonthlyFrequency:
	default:
		return errors.BadRequest("invalid frequency", nil)
	}
	if fl.Limit <= 0 {
		return errors.BadRequest("limit must be positive", nil)
	}
	return nil
}

// validateTimeWindow checks that, if both bounds are given, start precedes
// end.
func validateTimeWindow(start, end *time.Time) error {
	if start != nil && end != nil && !start.Before(*end) {
		return errors.BadRequest("daily_start_time must be before daily_end_time", nil)
	}
	return nil
}
