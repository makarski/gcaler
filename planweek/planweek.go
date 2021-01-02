package planweek

import (
	"context"
	"time"
)

// defaultDurationDays holds the default number of days
// to generate the planning dates for
const defaultDurationDays = 30

// Plan returns a channel of dates starting from startDate
// Each following date is 24h later than the previous one, i.e. the next day
func Plan(ctx context.Context, startDate time.Time, durationDays int) (<-chan time.Time, error) {
	if durationDays <= 0 {
		durationDays = defaultDurationDays
	}

	dates := make(chan time.Time)
	go func(currentDate time.Time, dayCount int) {
		for i := 0; i < dayCount; i++ {
			select {
			case <-ctx.Done():
				break
			default:
				dates <- currentDate
				currentDate = currentDate.Add(time.Hour * 24)
			}
		}
		close(dates)
	}(startDate, durationDays)

	return dates, nil
}
