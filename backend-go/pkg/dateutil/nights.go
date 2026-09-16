package dateutil

import (
	"fmt"
	"time"
)

const DateFormat = "2006-01-02"

// NightsBetween returns a slice of YYYY-MM-DD date strings for each night of a stay.
// Check-in 2026-09-01 to check-out 2026-09-03 yields ["2026-09-01", "2026-09-02"].
func NightsBetween(checkIn, checkOut string) ([]string, error) {
	if len(checkIn) < 10 || len(checkOut) < 10 {
		return nil, fmt.Errorf("invalid date format: checkIn=%q, checkOut=%q", checkIn, checkOut)
	}

	start, err := time.Parse(DateFormat, checkIn[:10])
	if err != nil {
		return nil, fmt.Errorf("failed to parse check-in date: %w", err)
	}

	end, err := time.Parse(DateFormat, checkOut[:10])
	if err != nil {
		return nil, fmt.Errorf("failed to parse check-out date: %w", err)
	}

	if !end.After(start) {
		return nil, fmt.Errorf("check-out date (%s) must be after check-in date (%s)", checkOut[:10], checkIn[:10])
	}

	var nights []string
	for cursor := start; cursor.Before(end); cursor = cursor.AddDate(0, 0, 1) {
		nights = append(nights, cursor.Format(DateFormat))
	}

	return nights, nil
}

// TodayISO returns today's date in UTC as YYYY-MM-DD
func TodayISO() string {
	return time.Now().UTC().Format(DateFormat)
}

// IsWeekendNight returns true if the night is a Friday or Saturday
func IsWeekendNight(t time.Time) bool {
	weekday := t.Weekday()
	return weekday == time.Friday || weekday == time.Saturday
}
