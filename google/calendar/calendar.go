package calendar

import (
	"context"

	"golang.org/x/oauth2"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"

	"github.com/makarski/gcaler/config"
	"github.com/makarski/gcaler/google/auth"
	"github.com/makarski/gcaler/staff"
)

const eventDateTimeFormat = "2006-01-02T15:04:05-07:00"

// GCalendar is a wrapper for Google Calendar Service
type GCalendar struct {
	gToken *auth.GToken
	cfg    *oauth2.Config
}

// NewGCalerndar inits a GCalendar struct
func NewGCalerndar(gToken *auth.GToken, cfg *oauth2.Config) GCalendar {
	return GCalendar{gToken, cfg}
}

// CalendarService inits a google calendar service
func (gc GCalendar) CalendarService(ctx context.Context, authHandler auth.ConsentHandlerFunc) (*calendar.Service, error) {
	tok, err := gc.gToken.Get(ctx, authHandler)
	if err != nil {
		return nil, err
	}

	return calendar.NewService(ctx, option.WithHTTPClient(gc.cfg.Client(ctx, tok)))
}

// CalendarEvent generates a google calendar event
func (gc GCalendar) CalendarEvent(
	a staff.Assignment,
	t *config.Template,
) (*calendar.Event, error) {
	startTime := a.Date.Format(eventDateTimeFormat)
	endTime := a.Date.Add(t.Duration).Format(eventDateTimeFormat)
	tzName, _ := a.Date.UTC().Zone()

	eRec, err := gcalEventRecurrence(&t.Recurrence)
	if err != nil {
		return nil, err
	}

	return &calendar.Event{
		Summary:     t.GenerateEventTitle(append(a.Assignees, &t.EventHost)...),
		Description: eventDescription(a.Assignees, t.Description),
		Start: &calendar.EventDateTime{
			DateTime: startTime,
			TimeZone: tzName,
		},
		End: &calendar.EventDateTime{
			DateTime: endTime,
			TimeZone: tzName,
		},
		Attendees:    eventAttendees(t.EventHost.Email, a.Assignees),
		Transparency: t.Transparency,
		Visibility:   t.Visibility,
		Recurrence:   eRec,
	}, nil
}

func eventAttendees(hostEmail string, atds []*config.Assignee) []*calendar.EventAttendee {
	attendees := []*calendar.EventAttendee{{Email: hostEmail, ResponseStatus: "accepted"}}
	for _, atd := range atds {
		attendees = append(attendees, &calendar.EventAttendee{Email: atd.Email, ResponseStatus: "needsAction"})
	}

	return attendees
}

func eventDescription(assignees []*config.Assignee, generic string) string {
	if len(assignees) != 1 {
		return generic
	}

	return assignees[0].Description
}

func gcalEventRecurrence(r *config.Recurrence) ([]string, error) {
	if r.Mode.IsSingle() {
		return nil, nil
	}

	return r.RFC5545()
}
