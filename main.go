package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/makarski/gcaler/cmd"
	"github.com/makarski/gcaler/config"
	"github.com/makarski/gcaler/google/auth"
	gcal "github.com/makarski/gcaler/google/calendar"
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

	// map of commands
	cmds = map[string]func(gcal.GCalendar, *config.Template) error{
		planCmd: cmd.Plan,
		listCmd: cmd.List,
		"":      cmd.Plan,
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
	if err := cmd(gCalendar, template); err != nil {
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
