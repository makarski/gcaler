package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/makarski/gcaler/config"
	"github.com/makarski/gcaler/google/auth"
	gcal "github.com/makarski/gcaler/google/calendar"
	"github.com/makarski/gcaler/staff"
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
	cfg, err := config.Load(configFile)
	if err != nil {
		panic(err)
	}

	if len(cfg.Templates) == 0 {
		fmt.Fprintln(os.Stdout, "No event templates found. Exit.")
		os.Exit(0)
	}

	var stdOutTemplate bytes.Buffer
	fmt.Fprintf(&stdOutTemplate, "> Select a template [0..%d]\n", len(cfg.Templates)-1)

	for i, template := range cfg.Templates {
		fmt.Fprintf(&stdOutTemplate, "  * %d: %s\n", i, template.Name)
	}

	templateChoice, err := stdIn(&stdOutTemplate)
	if err != nil {
		panic(err)
	}

	templateIndex, err := strconv.Atoi(templateChoice)
	if err != nil {
		panic(err)
	}

	template := cfg.Templates[templateIndex]

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
		staff.InputStrProviderFunc(stdIn),
		staff.InputBoolProviderFunc(stdInConfirm),
	)
	if err != nil {
		panic(err)
	}

	summary := summaryTxtBuffer(len(assignments))

	for _, assignment := range assignments {
		event := gCalendar.CalendarEvent(assignment, template.EventName, template.Duration)
		if _, err := calSrv.Events.Insert(template.CalID, event).Do(); err != nil {
			panic(err)
		}

		fmt.Fprintf(
			summary,
			"  * %s: %s\n",
			assignment.Assignee.FullName,
			assignment.Date.Format(time.RFC1123),
		)
	}

	if _, err := io.Copy(out, summary); err != nil {
		panic(err)
	}
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
	return stdIn(bytes.NewBufferString("> Enter auth. code: "))
}

func stdIn(buf io.ReadWriter) (string, error) {
	if _, err := io.Copy(os.Stdout, buf); err != nil {
		return "", err
	}

	var in string
	_, err := fmt.Fscanln(os.Stdin, &in)
	return in, err
}

func stdInConfirm(buf io.ReadWriter) (bool, error) {
	if _, err := buf.Write([]byte(" [y/n]: ")); err != nil {
		return false, err
	}

	if _, err := io.Copy(os.Stdout, buf); err != nil {
		return false, err
	}

	var in string
	if _, err := fmt.Fscanln(os.Stdin, &in); err != nil {
		return false, err
	}

	if in != "y" {
		return false, nil
	}

	return true, nil
}
