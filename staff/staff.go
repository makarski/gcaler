package staff

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
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

// Schedule returns a slice of Assignment pairs: Assignee to Date
func (a Assignees) Schedule(
	ctx context.Context,
	scanString InputStrProviderFunc,
	scanBool InputBoolProviderFunc,
) ([]Assignment, error) {
	startDate, err := scanString(bytes.NewBufferString("> Enter the kickoff date: "))
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	dates, err := planweek.Plan(ctx, startDate)
	if err != nil {
		return nil, err
	}

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
