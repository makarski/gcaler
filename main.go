package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"

	"github.com/makarski/gcaler/google/auth"
	"github.com/makarski/gcaler/planweek"
)

const appName = "gcaler"

var (
	tokenCacheDir  string
	tokenCacheFile string

	configFile      string
	credentialsFile string

	out = os.Stdout
	in  = os.Stdin
)

type (
	// Config struct describes the app config
	Config struct {
		CalID     string   `json:"cal_id"`
		StartTime string   `json:"start_time"`
		EndTime   string   `json:"end_time"`
		CtaText   string   `json:"cta_text"`
		People    []Person `json:"people"`
	}

	// Person describes a config Person entry
	Person struct {
		FullName string `json:"full_name"`
		Email    string `json:"email"`
		Phone    string `json:"phone"`
		Link     string `json:"link"`
	}
)

func init() {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	tokenCacheDir = filepath.Join(os.Getenv("HOME"), "."+appName)
	tokenCacheFile = filepath.Join(tokenCacheDir, "access_token.json")

	flag.StringVar(&configFile, "config", filepath.Join(wd, "config.json"), "Config file name: absolute or relative path")
	flag.StringVar(&credentialsFile, "credentials", filepath.Join(wd, "client_secret.json"), "Credentials file name: absolute or relative path")
	flag.Parse()
}

func main() {
	cfg, err := getConfig(configFile)
	if err != nil {
		panic(err)
	}

	credCfg, err := getCredentials(credentialsFile)
	if err != nil {
		panic(err)
	}

	gToken := auth.NewGToken(credCfg, tokenCacheFile, tokenCacheDir)
	calSrv, err := getCalendarService(context.Background(), gToken, credCfg)
	if err != nil {
		panic(err)
	}

	assignments, err := Assignees(cfg.People).schedule()
	if err != nil {
		panic(err)
	}

	summary := []string{}
	for _, assignment := range assignments {
		event := getEvent(Person(assignment.Assignee), assignment.Date, cfg.StartTime, cfg.EndTime)
		if _, err := calSrv.Events.Insert(cfg.CalID, event).Do(); err != nil {
			panic(err)
		}

		summary = append(
			summary,
			fmt.Sprintf(
				"> %s : %s (%s)",
				assignment.Assignee.FullName,
				assignment.Date.Format("2006-01-02"),
				assignment.Date.Weekday().String(),
			))
	}

	fmt.Fprintf(
		out,
		`
----------------------
assigned weekdays: %d
----------------------
`,
		len(summary),
	)

	for _, item := range summary {
		fmt.Fprintln(out, item)
	}
}

type (
	// Assignees is a list of people to be assigned to shifts
	Assignees []Person

	// Assignment contains a pair - Assigned Person and Date of the shift
	Assignment struct {
		Date     time.Time
		Assignee Person
	}
)

func (a Assignees) pick(i int) (*Person, error) {
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

func (a Assignees) schedule() ([]Assignment, error) {
	startDate, err := stdIn("> Enter the kickoff date: ")
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dates, err := planweek.Plan(ctx, startDate)
	if err != nil {
		return nil, err
	}

	assignments := make([]Assignment, 0)

	for date := range dates {
		pickCtaTxt := bytes.NewBufferString(fmt.Sprintf("> Pick up an assignee number for %s:\n\n", date.Format("2006-01-02")))
		a.print(pickCtaTxt)

		in, err := stdIn(pickCtaTxt.String())
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

		ok, err := stdInConfirm("> Do you want to continue assigning?")
		if err != nil {
			return nil, err
		}

		if !ok {
			break
		}
	}

	return assignments, nil
}

func handleAuthConsent(authURL string) (string, error) {
	fmt.Fprintf(out, "> Visit the link: %v\n", authURL)
	return stdIn("> Enter auth. code: ")
}

func getConfig(configFile string) (*Config, error) {
	b, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	var cfg Config
	return &cfg, json.Unmarshal(b, &cfg)
}

func getCredentials(credentialsFile string) (*oauth2.Config, error) {
	b, err := ioutil.ReadFile(credentialsFile)
	if err != nil {
		return nil, err
	}

	return google.ConfigFromJSON(b, calendar.CalendarScope)
}

func getCalendarService(ctx context.Context, gToken auth.GToken, cfg *oauth2.Config) (*calendar.Service, error) {
	tok, err := gToken.Get(ctx, handleAuthConsent)
	if err != nil {
		return nil, err
	}

	return calendar.New(cfg.Client(ctx, tok))
}

func getEvent(p Person, date time.Time, start, end string) *calendar.Event {
	startTime := date.Format("2006-01-02") + "T" + start
	endTime := date.Format("2006-01-02") + "T" + end

	return &calendar.Event{
		Summary:     fmt.Sprintf("On-Call: %s\n%s", p.FullName, p.Email),
		Description: fmt.Sprintf("phone: %s, link: %s", p.Phone, p.Link),
		Start: &calendar.EventDateTime{
			DateTime: startTime,
		},
		End: &calendar.EventDateTime{
			DateTime: endTime,
		},
		Attendees: []*calendar.EventAttendee{
			{Email: p.Email, ResponseStatus: "accepted"},
		},
		Transparency: "transparent",
	}
}

func stdIn(txt string) (string, error) {
	fmt.Fprint(os.Stdout, txt)
	var in string
	_, err := fmt.Fscanln(os.Stdin, &in)
	return in, err
}

func stdInConfirm(txt string) (bool, error) {
	fmt.Fprint(os.Stdout, txt+" [y/n]: ")
	var in string
	if _, err := fmt.Fscanln(os.Stdin, &in); err != nil {
		return false, err
	}

	if in != "y" {
		return false, nil
	}

	return true, nil
}
