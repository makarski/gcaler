package staff

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/makarski/gcaler/planweek"
)

type (
	// Assignees is a list of people to be assigned to shifts
	Assignees []Assignee

	// Assignee describes a config `people` item entry
	Assignee struct {
		FullName string `json:"full_name"`
		Email    string `json:"email"`
		Phone    string `json:"phone"`
		Link     string `json:"link"`
	}

	// Assignment contains a pair - Assigned Person and Date of the shift
	Assignment struct {
		Assignee
		Date time.Time
	}

	// InputStrProviderFunc is used to interactively control
	// the flow of the Schedule func
	InputStrProviderFunc func(out io.ReadWriter) (string, error)

	// InputBoolProviderFunc is used to interactively control
	// the flow of the Schedule func
	InputBoolProviderFunc func(out io.ReadWriter) (bool, error)
)

func (a Assignees) pick(i int) (*Assignee, error) {
	if i > len(a)-1 {
		return nil, fmt.Errorf("no assignee found by index: %d", i)
	}
	return &a[i], nil
}

func (a Assignees) print(w io.Writer) {
	for i, person := range a {
		fmt.Fprintf(w, "  > %d: %s\n", i, person.FullName)
	}
}

func (a Assignees) startDateAndDuration(scanString InputStrProviderFunc) (*time.Time, int, error) {
	startDate, err := scanString(bytes.NewBufferString("> Enter a kickoff date: "))
	if err != nil {
		return nil, 0, err
	}

	date, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return nil, 0, err
	}

	durationDays, err := scanString(bytes.NewBufferString("> Enter a number of days to schedule ('-1' to proceed day by day): "))
	if err != nil {
		return nil, 0, err
	}

	days, err := strconv.Atoi(durationDays)
	if err != nil {
		return nil, 0, err
	}

	return &date, days, nil
}

// Schedule returns a slice of Assignment pairs: Assignee to Date
func (a Assignees) Schedule(
	ctx context.Context,
	scanString InputStrProviderFunc,
	scanBool InputBoolProviderFunc,
) ([]Assignment, error) {
	startDate, durationDays, err := a.startDateAndDuration(scanString)
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
		return a.assignBatch(dates, scanString)
	}

	return a.assignOneByOne(dates, scanString, scanBool)
}

func (a Assignees) assignOneByOne(
	dates <-chan time.Time,
	scanString InputStrProviderFunc,
	scanBool InputBoolProviderFunc,
) ([]Assignment, error) {
	assignments := make([]Assignment, 0)
	for date := range dates {
		var pickCtaTxt bytes.Buffer
		_, err := fmt.Fprintf(&pickCtaTxt, "> Pick up an assignee number for %s:\n\n", date.Format("2006-01-02"))
		if err != nil {
			return nil, err
		}
		a.print(&pickCtaTxt)

		in, err := scanString(&pickCtaTxt)
		if err != nil {
			return nil, err
		}

		pickedIndex, err := strconv.Atoi(in)
		if err != nil {
			return nil, err
		}

		assignedPerson, err := a.pick(pickedIndex)
		if err != nil {
			return nil, err
		}

		assignment := Assignment{Date: date, Assignee: *assignedPerson}
		assignments = append(assignments, assignment)

		ok, err := scanBool(bytes.NewBufferString("> Do you want to continue assigning?"))
		if err != nil {
			return nil, err
		}

		if !ok {
			break
		}
	}

	return assignments, nil
}

func (a Assignees) assignBatch(dates <-chan time.Time, scanString InputStrProviderFunc) ([]Assignment, error) {
	assignments := make([]Assignment, 0)

	var pickCtaTxt bytes.Buffer
	_, err := fmt.Fprintf(&pickCtaTxt, "> Available Assignees:\n")
	if err != nil {
		return nil, err
	}

	a.print(&pickCtaTxt)
	pickCtaTxt.WriteString("\n")

	_, err = fmt.Fprintf(&pickCtaTxt, "> Dates to schedule:\n")
	if err != nil {
		return nil, err
	}

	schedule := make([]time.Time, 0)

	for date := range dates {
		schedule = append(schedule, date)
		fmt.Fprintln(&pickCtaTxt, "  >", date.Format("2006-01-02 (Mon)"))
	}

	pickCtaTxt.WriteString("\n> Enter Assignee Sequence by Number (without spaces. ex: 0,1,1,0): ")
	in, err := scanString(&pickCtaTxt)
	if err != nil {
		return nil, err
	}

	inPicks := strings.Split(strings.TrimSpace(in), ",")
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
