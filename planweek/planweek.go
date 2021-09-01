package planweek

import (
	"context"
	"time"
)

// Plan returns a channel of dates starting from startDate
func Plan(ctx context.Context, startDate time.Time, eventCount int32, interval time.Duration) (<-chan time.Time, error) {
	dates := make(chan time.Time)
	go func(currentDate time.Time, eventCount int32) {
	out:
		for i := 0; i < int(eventCount); i++ {
			select {
			case <-ctx.Done():
				break out
			default:
				dates <- currentDate
				currentDate = currentDate.Add(interval)
			}
		}
		close(dates)
	}(startDate, eventCount)

	return dates, nil
}
