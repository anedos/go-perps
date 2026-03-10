package api

import (
	"fmt"
	"strconv"
	"time"
)

func parsePeriod(value string, now time.Time) (time.Time, error) {
	if value == "" {
		return time.Time{}, fmt.Errorf("period is required")
	}

	if duration, err := time.ParseDuration(value); err == nil {
		return now.Add(-duration), nil
	}

	if len(value) < 2 {
		return time.Time{}, fmt.Errorf("invalid period %q", value)
	}

	amount, err := strconv.Atoi(value[:len(value)-1])
	if err != nil || amount <= 0 {
		return time.Time{}, fmt.Errorf("invalid period %q", value)
	}

	switch value[len(value)-1] {
	case 'D', 'd':
		return now.AddDate(0, 0, -amount), nil
	case 'W', 'w':
		return now.AddDate(0, 0, -7*amount), nil
	case 'M':
		return now.AddDate(0, -amount, 0), nil
	default:
		return time.Time{}, fmt.Errorf("invalid period %q", value)
	}
}
