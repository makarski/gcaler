package calendar

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/calendar/v3"

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
func (gc GCalendar) CalendarEvent(a staff.Assignment, eventName string, duration time.Duration) *calendar.Event {
	startTime := a.Date.Format(eventDateTimeFormat)
	endTime := a.Date.Add(duration).Format(eventDateTimeFormat)

	return &calendar.Event{
		Summary:     fmt.Sprintf("%s: %s", eventName, a.Email),
		Description: fmt.Sprintf("phone: %s, link: %s", a.Phone, a.Link),
		Start: &calendar.EventDateTime{
			DateTime: startTime,
		},
		End: &calendar.EventDateTime{
			DateTime: endTime,
		},
		Attendees: []*calendar.EventAttendee{
			{Email: a.Email, ResponseStatus: "accepted"},
		},
		Transparency: "transparent",
	}
}
