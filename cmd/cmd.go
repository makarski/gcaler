package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/makarski/gcaler/config"
	gcal "github.com/makarski/gcaler/google/calendar"
	"github.com/makarski/gcaler/staff"
	"github.com/makarski/gcaler/userio"
)

var out = os.Stdout

func List(_ gcal.GCalendar, _ *config.Template) {
	fmt.Println("not implemented")
}

func Plan(gCalendar gcal.GCalendar, template *config.Template) {
	ctx := context.Background()
	calSrv, err := gCalendar.CalendarService(ctx, handleAuthConsent)
	if err != nil {
		panic(err)
	}

	tz, err := time.LoadLocation(template.Timezone)
	if err != nil {
		panic(err)
	}

	assignments, err := staff.Assignees(template.Participants).Schedule(
		ctx,
		tz,
		&template.Recurrence,
	)
	if err != nil {
		panic(err)
	}

	summary := summaryTxtBuffer(len(assignments))

	for _, assignment := range assignments {
		event, err := gCalendar.CalendarEvent(
			assignment,
			template,
		)
		if err != nil {
			panic(err)
		}

		if _, err := calSrv.Events.Insert(template.CalID, event).Do(); err != nil {
			panic(err)
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

	if _, err := io.Copy(out, summary); err != nil {
		panic(err)
	}
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
