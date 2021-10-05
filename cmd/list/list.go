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

	statusCancelled   = "cancelled"
	statusAccepted    = "accepted"
	statusNeedsAction = "needsAction"
	statusDeclined    = "declined"
	statusTentative   = "tentative"

	eventHost      = "host"
	eventTypeVideo = "video"
)

var (
	emojiStatus = map[string]string{
		statusNeedsAction: whiteSquare,
		statusAccepted:    checkMark,
		statusDeclined:    crossMark,
		statusTentative:   questionMark,
		eventHost:         ship,
	}
)

func List(email string) cmd.CmdFunc {
	template := loadTemplate(email)
	return func(gCalendar gcal.GCalendar) error {
		ctx := context.Background()
		calSrv, tz, err := cmd.CalSrvLocation(ctx, &gCalendar, template)
		if err != nil {
			return err
		}

		now := time.Now()
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, tz)
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
			if !cursorSet && isAcceptedOrTentative(event, template.CalID) {
				nowCursor, cursorSet, err = scheduleCursorEmoji(now, startEnd[0], startEnd[1])
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

			if event.Location != "" {
				fmt.Printf("    %s\n", event.Location)
			}
		}

		return nil
	}
}

func loadTemplate(email string) *config.Template {
	return &config.Template{CalID: email}
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

func scheduleCursorEmoji(reference time.Time, eventStart, eventEnd time.Time) (string, bool, error) {
	isNow := reference.After(eventStart) && reference.Before(eventEnd)
	isNext := reference.Before(eventStart)

	if isNow || isNext {
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

func isAcceptedOrTentative(event *calendar.Event, email string) bool {
	status := eventResponseStatus(event, email)
	return status == statusAccepted || status == statusTentative || status == eventHost
}
