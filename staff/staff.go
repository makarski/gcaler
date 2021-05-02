package staff

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/makarski/gcaler/planweek"
	"github.com/makarski/gcaler/userio"
)

type (
	// Assignees is a list of people to be assigned to shifts
	Assignees []Assignee

	// Assignee describes a config `people` item entry
	Assignee struct {
		FullName    string `toml:"full_name"`
		Email       string `toml:"email"`
		Description string `toml:"description"`
	}

	// Assignment contains a pair - Assigned Person and Date of the shift
	Assignment struct {
		Assignee
		Date time.Time
	}
)

func (a Assignees) pick(i int) (*Assignee, error) {
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

func (a Assignees) startDateAndDuration(cfgStartTime string) (*time.Time, int, error) {
	startDate, err := userio.UserIn(bytes.NewBufferString("> Enter a kickoff date: "))
	if err != nil {
		return nil, 0, err
	}

	date, err := time.Parse("2006-01-02T15:04:05-07:00", startDate+"T"+cfgStartTime)
	if err != nil {
		return nil, 0, err
	}

	days, err := userio.UserInInt(bytes.NewBufferString("> Enter a number of days to schedule ('-1' to proceed day by day): "))
	if err != nil {
		return nil, 0, err
	}

	return &date, days, nil
}

// Schedule returns a slice of Assignment pairs: Assignee to Date
func (a Assignees) Schedule(
	ctx context.Context,
	cfgStartTimeTZ string,
) ([]Assignment, error) {
	startDate, durationDays, err := a.startDateAndDuration(cfgStartTimeTZ)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	dates, err := planweek.Plan(ctx, *startDate, durationDays)
	if err != nil {
		return nil, err
	}

	if durationDays > 0 {
		return a.assignBatch(dates)
	}

	return a.assignOneByOne(dates)
}

func (a Assignees) assignOneByOne(
	dates <-chan time.Time,
) ([]Assignment, error) {
	assignments := make([]Assignment, 0)
	for date := range dates {
		var pickCtaTxt bytes.Buffer
		_, err := fmt.Fprintf(&pickCtaTxt, "> Pick up an assignee number for %s:\n\n", date.Format("2006-01-02"))
		if err != nil {
			return nil, err
		}
		a.print(&pickCtaTxt)

		pickedIndex, err := userio.UserInInt(&pickCtaTxt)
		if err != nil {
			return nil, err
		}

		assignedPerson, err := a.pick(pickedIndex)
		if err != nil {
			return nil, err
		}

		assignment := Assignment{Date: date, Assignee: *assignedPerson}
		assignments = append(assignments, assignment)

		ok, err := userio.UserInBool(bytes.NewBufferString("> Do you want to continue assigning?"))
		if err != nil {
			return nil, err
		}

		if !ok {
			break
		}
	}

	return assignments, nil
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
