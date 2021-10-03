package plan

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/makarski/gcaler/cmd"
	"github.com/makarski/gcaler/config"
	gcal "github.com/makarski/gcaler/google/calendar"
	"github.com/makarski/gcaler/staff"
	"github.com/makarski/gcaler/userio"
)

func Plan(templatesDir string) cmd.CmdFunc {
	template, err := loadTemplate(templatesDir)
	return func(gCalendar gcal.GCalendar) error {
		if err != nil {
			return err
		}

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

func loadTemplate(templatesDir string) (*config.Template, error) {
	templateCfgs, err := os.ReadDir(templatesDir)
	if err != nil {
		return nil, err
	}

	if len(templateCfgs) == 0 {
		fmt.Fprintln(os.Stdout, "No event templates found. Exit.")
		os.Exit(0)
	}

	var stdOutTemplate bytes.Buffer
	fmt.Fprintf(&stdOutTemplate, "> Select a template [0..%d]\n", len(templateCfgs)-1)

	for i, templateFile := range templateCfgs {
		fmt.Fprintf(&stdOutTemplate, "  * %d: %s\n", i, templateFile.Name())
	}

	stdOutTemplate.WriteString("\n> Template: ")

	templateIndex, err := userio.UserInInt(&stdOutTemplate)
	if err != nil {
		return nil, err
	}

	templateFile := filepath.Join(templatesDir, templateCfgs[templateIndex].Name())
	return config.LoadTemplate(templateFile)
}
