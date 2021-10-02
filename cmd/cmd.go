package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"google.golang.org/api/calendar/v3"

	"github.com/makarski/gcaler/config"
	gcal "github.com/makarski/gcaler/google/calendar"
	"github.com/makarski/gcaler/staff"
	"github.com/makarski/gcaler/userio"
)

var out = os.Stdout

func Plan(gCalendar gcal.GCalendar, template *config.Template) error {
	ctx := context.Background()
	calSrv, tz, err := calSrvLocation(ctx, &gCalendar, template)
	if err != nil {
		return err
	}

	assignments, err := staff.Assignees(template.Participants).Schedule(
		ctx,
		tz,
		&template.Recurrence,
	)
	if err != nil {
		return err
	}

	summary := summaryTxtBuffer(len(assignments))

	for _, assignment := range assignments {
		event, err := gCalendar.CalendarEvent(
			assignment,
			template,
		)
		if err != nil {
			return err
		}

		if _, err := calSrv.Events.Insert(template.CalID, event).Do(); err != nil {
			return err
		}

		for _, asgnee := range assignment.Assignees {
			fmt.Fprintf(
				summary,
				"  * %s: %s\n",
				asgnee.FullName(),
				assignment.Date.Format(time.RFC1123),
			)
		}
	}

	_, err = io.Copy(out, summary)
	return err
}

func calSrvLocation(
	ctx context.Context,
	gCalendar *gcal.GCalendar,
	template *config.Template,
) (*calendar.Service, *time.Location, error) {
	calSrv, err := gCalendar.CalendarService(ctx, handleAuthConsent)
	if err != nil {
		return nil, nil, err
	}

	tz, err := time.LoadLocation(template.Timezone)
	if err != nil {
		return nil, nil, err
	}

	return calSrv, tz, nil
}

func summaryTxtBuffer(countAssgnmts int) *bytes.Buffer {
	var summary bytes.Buffer
	fmt.Fprintf(
		&summary,
		`
Events created: %d
-------------------
`,
		countAssgnmts,
	)
	return &summary
}

func handleAuthConsent(authURL string) (string, error) {
	fmt.Fprintf(out, "> Visit the link: %v\n", authURL)
	return userio.UserIn(bytes.NewBufferString("> Enter auth. code: "))
}
