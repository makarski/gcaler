package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/makarski/gcaler/config"
	"github.com/makarski/gcaler/google/auth"
	gcal "github.com/makarski/gcaler/google/calendar"
	"github.com/makarski/gcaler/staff"
	"github.com/makarski/gcaler/userio"
)

const (
	appName = "gcaler"
	planCmd = "plan"
	listCmd = "list"
)

var (
	fls *flag.FlagSet

	tokenCacheDir  string
	tokenCacheFile string

	templatesDir    string
	credentialsFile string

	out = os.Stdout

	// map of commands
	cmds = map[string]func(gcal.GCalendar, *config.Template){
		planCmd: plan,
		listCmd: list,
		"":      plan,
	}
)

func init() {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	tokenCacheDir = filepath.Join(os.Getenv("HOME"), "."+appName)
	tokenCacheFile = filepath.Join(tokenCacheDir, "access_token.json")

	fls = flag.NewFlagSet("", flag.ExitOnError)

	fls.StringVar(&templatesDir, "templates", filepath.Join(wd, "templates"), "Path to templates directory")
	fls.StringVar(&credentialsFile, "credentials", filepath.Join(wd, "client_secret.json"), "Credentials file name: absolute or relative path")

	fls.Usage = printHelp
}

func printHelp() {
	txt := `
Calendar planner

USAGE:
  gcaler [OPTIONS] [SUBCOMMAND]

SUBCOMMANDS:
  plan		Schedule an based on the template config
  list		List calendar events

OPTIONS:
`

	fmt.Fprint(fls.Output(), txt)
	fls.PrintDefaults()
}

func main() {
	// parse flags
	fls.Parse(os.Args[1:])

	// parse subcommand
	cmdName := fls.Arg(0)
	cmd, ok := cmds[cmdName]
	if !ok {
		panic(fmt.Sprintf("cmd: `%s` not found", cmdName))
	}

	template, err := loadTemplate()
	if err != nil {
		panic(err)
	}

	gToken := auth.NewGToken(credentialsFile, tokenCacheFile, tokenCacheDir)
	credCfg, err := gToken.Credentials()
	if err != nil {
		panic(err)
	}

	gCalendar := gcal.NewGCalerndar(&gToken, credCfg)

	// execute
	cmd(gCalendar, template)
}

func list(_ gcal.GCalendar, _ *config.Template) {
	fmt.Println("not implemented")
}

func plan(gCalendar gcal.GCalendar, template *config.Template) {
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

func loadTemplate() (*config.Template, error) {
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

func summaryTxtBuffer(countAssgnmts int) *bytes.Buffer {
	var summary bytes.Buffer
	fmt.Fprintf(
		&summary,
		`
----------------------
assigned weekdays: %d
----------------------
`,
		countAssgnmts,
	)
	return &summary
}

func handleAuthConsent(authURL string) (string, error) {
	fmt.Fprintf(out, "> Visit the link: %v\n", authURL)
	return userio.UserIn(bytes.NewBufferString("> Enter auth. code: "))
}
