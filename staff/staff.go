package staff

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/makarski/gcaler/config"
	"github.com/makarski/gcaler/planweek"
	"github.com/makarski/gcaler/userio"
)

type (
	// Assignees is a list of people to be assigned to shifts
	Assignees []config.Assignee

	// Assignment contains a pair - Assigned Person and Date of the shift
	Assignment struct {
		config.Assignee
		Date time.Time
	}
)

func (a Assignees) pick(i int) (*config.Assignee, error) {
	if i > len(a)-1 {
		return nil, fmt.Errorf("no assignee found by index: %d", i)
	}
	return &a[i], nil
}

func (a Assignees) print(w io.Writer) {
	for i, person := range a {
		fmt.Fprintf(w, "  * %d: %s\n", i, person.FullName)
	}
}

func (a Assignees) startDate(cfgStartTime string) (*time.Time, error) {
	startDate, err := userio.UserIn(bytes.NewBufferString("> Enter a kickoff date: "))
	if err != nil {
		return nil, err
	}

	date, err := time.Parse("2006-01-02T15:04:05-07:00", startDate+"T"+cfgStartTime)
	if err != nil {
		return nil, err
	}

	return &date, nil
}

// Schedule returns a slice of Assignment pairs: Assignee to Date
func (a Assignees) Schedule(
	ctx context.Context,
	cfgStartTimeTZ string,
	recurrence *config.Recurrence,
) ([]Assignment, error) {
	startDate, err := a.startDate(cfgStartTimeTZ)
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
	inPicks := []string{}

	for date := range dates {
		schedule = append(schedule, date)
		fmt.Fprint(&pickCtaTxt, "  * ", date.Format("2006-01-02 (Mon): "))
		in, err := userio.UserIn(&pickCtaTxt)
		if err != nil {
			return nil, err
		}

		inPicks = append(inPicks, in)
	}

	for i, inPick := range inPicks {
		pickedIndex, err := strconv.Atoi(inPick)
		if err != nil {
			return nil, err
		}

		assignedPerson, err := a.pick(pickedIndex)
		if err != nil {
			return nil, err
		}

		assignment := Assignment{Date: schedule[i], Assignee: *assignedPerson}
		assignments = append(assignments, assignment)
	}

	return assignments, nil
}
