package staff

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/makarski/gcaler/config"
	"github.com/makarski/gcaler/planweek"
	"github.com/makarski/gcaler/userio"
)

type (
	// Assignees is a list of people to be assigned to shifts
	Assignees []*config.Assignee

	// Assignment contains a pair - Assigned Person and Date of the shift
	Assignment struct {
		Assignees
		Date time.Time
	}
)

func (a Assignees) pick(i int) (*config.Assignee, error) {
	if i > len(a)-1 {
		return nil, fmt.Errorf("no assignee found by index: %d", i)
	}
	return a[i], nil
}

func (a Assignees) print(w io.Writer) {
	for i, person := range a {
		fmt.Fprintf(w, "  * %d: %s\n", i, person.FullName())
	}
}

func (a Assignees) startDate(timezone *time.Location) (*time.Time, error) {
	startDate, err := userio.UserIn(bytes.NewBufferString("> Enter event date (ex: 2006-10-22): "))
	if err != nil {
		return nil, err
	}

	startTime, err := userio.UserIn(bytes.NewBufferString("> Enter event time (ex: 15:04): "))
	if err != nil {
		return nil, err
	}

	date, err := time.ParseInLocation("2006-01-02 15:04", startDate+" "+startTime, timezone)
	if err != nil {
		return nil, err
	}

	return &date, nil
}

// Schedule returns a slice of Assignment pairs: Assignee to Date
func (a Assignees) Schedule(
	ctx context.Context,
	timezone *time.Location,
	recurrence *config.Recurrence,
) ([]Assignment, error) {
	startDate, err := a.startDate(timezone)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	eventCount := recurrence.Count
	if recurrence.Mode.IsRecurrent() {
		eventCount = 1
	}

	dates, err := planweek.Plan(ctx, *startDate, eventCount, recurrence.Frequency)
	if err != nil {
		return nil, err
	}

	return a.assignBatch(dates)
}

func (a Assignees) assignBatch(dates <-chan time.Time) ([]Assignment, error) {
	assignments := make([]Assignment, 0)

	var pickCtaTxt bytes.Buffer
	_, err := fmt.Fprintf(&pickCtaTxt, "> Available Assignees:\n")
	if err != nil {
		return nil, err
	}

	a.print(&pickCtaTxt)
	pickCtaTxt.WriteString("\n")

	_, err = fmt.Fprintf(&pickCtaTxt, "> Enter an Assignee for a Date [0..%d]:\n", len(a)-1)
	if err != nil {
		return nil, err
	}

	schedule := make([]time.Time, 0)
	inPicks := make([][]string, 0)

	for date := range dates {
		schedule = append(schedule, date)
		fmt.Fprint(&pickCtaTxt, "  * ", date.Format("2006-01-02 (Mon): "))
		in, err := userio.UserIn(&pickCtaTxt)
		if err != nil {
			return nil, err
		}

		inPicks = append(inPicks, strings.Split(in, " "))
	}

	for i, inPick := range inPicks {
		assignees := make([]*config.Assignee, 0, len(inPicks))
		for _, pick := range inPick {
			pickedIndex, err := strconv.Atoi(pick)
			if err != nil {
				return nil, err
			}

			assignedPerson, err := a.pick(pickedIndex)
			if err != nil {
				return nil, err
			}

			assignees = append(assignees, assignedPerson)

		}
		assignment := Assignment{Date: schedule[i], Assignees: assignees}
		assignments = append(assignments, assignment)
	}

	return assignments, nil
}
