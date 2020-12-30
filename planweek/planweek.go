package planweek

import (
	"context"
	"time"
)

// Plan returns a channel of dates starting from startDate
// Each following date is 24h later than the previous one, i.e. the next day
func Plan(ctx context.Context, startDate string) (<-chan time.Time, error) {
	date, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return nil, err
	}

	dates := make(chan time.Time)
	go func(currentDate time.Time) {
		for {
			select {
			case <-ctx.Done():
				close(dates)
				break
			default:
				dates <- currentDate
				currentDate = currentDate.Add(time.Hour * 24)
			}
		}
	}(date)

	return dates, nil
}
