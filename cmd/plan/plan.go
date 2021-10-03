package plan

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/makarski/gcaler/cmd"
	"github.com/makarski/gcaler/config"
	gcal "github.com/makarski/gcaler/google/calendar"
	"github.com/makarski/gcaler/staff"
)

func Plan(gCalendar gcal.GCalendar, template *config.Template) error {
	ctx := context.Background()
	calSrv, tz, err := cmd.CalSrvLocation(ctx, &gCalendar, template)
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

	_, err = io.Copy(cmd.Out, summary)
	return err
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
