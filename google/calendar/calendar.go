package calendar

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/calendar/v3"

	"github.com/makarski/gcaler/google/auth"
)

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

// Assignee describes a config `people` item entry
type Assignee struct {
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Link     string `json:"link"`
}

// CalendarEvent generates a google calendar event
func (gc GCalendar) CalendarEvent(a Assignee, date time.Time, start, end string) *calendar.Event {
	startTime := date.Format("2006-01-02") + "T" + start
	endTime := date.Format("2006-01-02") + "T" + end

	return &calendar.Event{
		Summary:     fmt.Sprintf("On-Call: %s\n%s", a.FullName, a.Email),
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
