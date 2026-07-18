package blocklist

import (
	"time"

	"github.com/insanelyharsh/hontest-habit/internal/common/errors"
	"github.com/insanelyharsh/hontest-habit/internal/constants"
	"github.com/insanelyharsh/hontest-habit/internal/types"
)

// computePeriodBounds returns the [start, end) window, in UTC, of the
// current period for freq containing now.
//
//   - DailyFrequency:   the calendar day containing now (UTC midnight to
//     next UTC midnight).
//   - WeeklyFrequency:  the ISO-8601 week containing now — Monday 00:00 UTC
//     through the following Monday 00:00 UTC.
//   - MonthlyFrequency: the calendar month containing now — the 1st at
//     00:00 UTC through the 1st of the next month at 00:00 UTC.
func computePeriodBounds(freq types.Frequency, now time.Time) (start, end time.Time, err error) {
	now = now.UTC()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	switch freq {
	case constants.DailyFrequency:
		return dayStart, dayStart.AddDate(0, 0, 1), nil

	case constants.WeeklyFrequency:
		// time.Weekday: Sunday=0 ... Saturday=6. Convert to a
		// Monday-start offset: Monday=0 ... Sunday=6.
		offset := (int(dayStart.Weekday()) + 6) % 7
		weekStart := dayStart.AddDate(0, 0, -offset)
		return weekStart, weekStart.AddDate(0, 0, 7), nil

	case constants.MonthlyFrequency:
		monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		return monthStart, monthStart.AddDate(0, 1, 0), nil

	default:
		return time.Time{}, time.Time{}, errors.Internal("unknown frequency", nil)
	}
}
