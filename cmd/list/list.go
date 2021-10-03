package list

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"google.golang.org/api/calendar/v3"

	"github.com/makarski/gcaler/cmd"
	"github.com/makarski/gcaler/config"
	gcal "github.com/makarski/gcaler/google/calendar"
)

const (
	ship         = "\\U+1F6A2"
	checkMark    = "\\U+2705"
	questionMark = "\\U+2753"
	crossMark    = "\\U+274C"
	clock        = "\\U+1F55C"
	whiteSquare  = "\\U+25FD"

	statusCancelled = "cancelled"
	eventHost       = "host"
	eventTypeVideo  = "video"
)

var (
	emojiStatus = map[string]string{
		"needsAction": whiteSquare,
		"accepted":    checkMark,
		"declined":    crossMark,
		"tentative":   questionMark,
		"host":        ship,
	}
)

func List(gCalendar gcal.GCalendar, template *config.Template) error {
	ctx := context.Background()
	calSrv, tz, err := cmd.CalSrvLocation(ctx, &gCalendar, template)
	if err != nil {
		return err
	}

	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, tz)
	end := start.Add(time.Hour * 24)

	events, err := calSrv.Events.
		List(template.CalID).
		TimeMin(start.Format(time.RFC3339)).
		TimeMax(end.Format(time.RFC3339)).
		SingleEvents(true).
		OrderBy("startTime").
		Do()

	if err != nil {
		return err
	}

	cursorSet := false

	for _, event := range events.Items {
		if event.Status == statusCancelled {
			continue
		}

		startEnd, err := parseEventTime(event)
		if err != nil {
			return err
		}

		nowCursor := "  "
		if !cursorSet {
			nowCursor, cursorSet, err = scheduleCursorEmoji(now, startEnd[0])
			if err != nil {
				return err
			}
		}

		ownResponse := eventResponseStatus(event, template.CalID)
		emoji, err := statusEmoji(ownResponse)
		if err != nil {
			return err
		}

		fmt.Printf("\n%s %s %s - %s: %s\n",
			nowCursor,
			emoji,
			startEnd[0].Format(time.Kitchen),
			startEnd[1].Format(time.Kitchen),
			event.Summary,
		)

		if event.ConferenceData != nil {
			for _, conf := range event.ConferenceData.EntryPoints {
				if conf.EntryPointType == eventTypeVideo {
					fmt.Printf("    %s\n", conf.Uri)
				}
			}
		}

	}

	return nil
}

func eventResponseStatus(event *calendar.Event, email string) string {
	if event.Creator.Email == email {
		return eventHost
	}

	for _, attendee := range event.Attendees {
		if attendee.Email == email {
			return attendee.ResponseStatus
		}
	}

	return ""
}

func parseEventTime(event *calendar.Event) ([]time.Time, error) {
	startEnd := make([]time.Time, 0, 2)
	for _, dateTime := range []*calendar.EventDateTime{event.Start, event.End} {
		parsed, err := time.Parse(time.RFC3339, dateTime.DateTime)
		if err != nil {
			return nil, err
		}
		startEnd = append(startEnd, parsed)
	}

	return startEnd, nil
}

func statusEmoji(status string) (string, error) {
	code, ok := emojiStatus[status]
	if !ok {
		return "*", nil
	}

	return emojiFromCode(code)
}

func scheduleCursorEmoji(reference time.Time, eventStart time.Time) (string, bool, error) {
	if reference.Before(eventStart) {
		e, err := emojiFromCode(clock)
		return e, true, err
	}

	return "  ", false, nil
}

func emojiFromCode(code string) (string, error) {
	emoji, err := strconv.ParseInt(strings.TrimPrefix(code, "\\U"), 16, 32)
	if err != nil {
		return "", err
	}

	return string(rune(emoji)), nil
}
