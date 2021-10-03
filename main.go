package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/makarski/gcaler/cmd"
	"github.com/makarski/gcaler/cmd/list"
	"github.com/makarski/gcaler/cmd/plan"
	"github.com/makarski/gcaler/google/auth"
	gcal "github.com/makarski/gcaler/google/calendar"
)

const (
	appName     = "gcaler"
	planCmdName = "plan"
	listCmdName = "list"
)

var (
	fls *flag.FlagSet

	tokenCacheDir  string
	tokenCacheFile string

	templatesDir    string
	credentialsFile string
	calId           string
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
	fls.StringVar(&calId, "email", calId, "Optional: email (calendar id) - used for 'list' subcmd")

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

	cmdRun, err := func() (cmd.CmdFunc, error) {
		switch cmdName {
		case planCmdName:
			return plan.Plan(templatesDir), nil
		case listCmdName, "":
			if calId == "" {
				return nil, fmt.Errorf("`-email` option must be provided")
			}

			return list.List(calId), nil
		}
		return nil, fmt.Errorf("cmd: `%s` not found", cmdName)
	}()

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
	if err := cmdRun(gCalendar); err != nil {
		panic(err)
	}
}
