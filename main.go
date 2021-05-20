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

const appName = "gcaler"

var (
	tokenCacheDir  string
	tokenCacheFile string

	templatesDir    string
	credentialsFile string

	out = os.Stdout
)

func init() {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	tokenCacheDir = filepath.Join(os.Getenv("HOME"), "."+appName)
	tokenCacheFile = filepath.Join(tokenCacheDir, "access_token.json")

	flag.StringVar(&templatesDir, "templates", filepath.Join(wd, "templates"), "Path to templates directory")
	flag.StringVar(&credentialsFile, "credentials", filepath.Join(wd, "client_secret.json"), "Credentials file name: absolute or relative path")
	flag.Parse()
}

func main() {
	templateCfgs, err := os.ReadDir(templatesDir)
	if err != nil {
		panic(err)
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
		panic(err)
	}

	template, err := loadTemplate(templateCfgs[templateIndex].Name())
	if err != nil {
		panic(err)
	}

	gToken := auth.NewGToken(credentialsFile, tokenCacheFile, tokenCacheDir)
	credCfg, err := gToken.Credentials()
	if err != nil {
		panic(err)
	}

	gCalendar := gcal.NewGCalerndar(&gToken, credCfg)

	ctx := context.Background()
	calSrv, err := gCalendar.CalendarService(ctx, handleAuthConsent)
	if err != nil {
		panic(err)
	}

	assignments, err := staff.Assignees(template.Participants).Schedule(
		ctx,
		template.StartTimeTZ,
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

		fmt.Fprintf(
			summary,
			"  * %s: %s\n",
			assignment.Assignee.FullName(),
			assignment.Date.Format(time.RFC1123),
		)
	}

	if _, err := io.Copy(out, summary); err != nil {
		panic(err)
	}
}

func loadTemplate(name string) (*config.Template, error) {
	templateFile := filepath.Join(templatesDir, name)
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
