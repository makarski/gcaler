package calendar

import (
	"context"

	"golang.org/x/oauth2"
	"google.golang.org/api/calendar/v3"

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

	return calendar.New(gc.cfg.Client(ctx, tok))
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
		Summary:     t.GenerateEventTitle(a.Assignee, t.EventHost),
		Description: a.Description,
		Start: &calendar.EventDateTime{
			DateTime: startTime,
			TimeZone: tzName,
		},
		End: &calendar.EventDateTime{
			DateTime: endTime,
			TimeZone: tzName,
		},
		Attendees: []*calendar.EventAttendee{
			{Email: t.EventHost.Email, ResponseStatus: "accepted"},
			{Email: a.Email, ResponseStatus: "needsAction"},
		},
		Transparency: "transparent",
		Recurrence:   eRec,
	}, nil
}

func gcalEventRecurrence(r *config.Recurrence) ([]string, error) {
	if r.Mode.IsSingle() {
		return nil, nil
	}

	return r.RFC5545()
}
