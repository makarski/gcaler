package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

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

	assignments, err := staff.Assignees(cfg.People).Schedule(
		ctx,
		staff.InputStrProviderFunc(stdIn),
		staff.InputBoolProviderFunc(stdInConfirm),
	)
	if err != nil {
		panic(err)
	}

	summary := summaryTxtBuffer(len(assignments))

	for _, assignment := range assignments {
		event := gCalendar.CalendarEvent(assignment, cfg.StartTime, cfg.EndTime)
		if _, err := calSrv.Events.Insert(cfg.CalID, event).Do(); err != nil {
			panic(err)
		}

		fmt.Fprintf(
			summary,
			"> %s : %s (%s)\n",
			assignment.Assignee.FullName,
			assignment.Date.Format("2006-01-02"),
			assignment.Date.Weekday().String(),
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
