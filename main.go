package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
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
	weekdays, err := getWeekdays()
	if err != nil {
		panic(err)
	}

	if weekdays == nil {
		return
	}

	cfg, err := getConfig(configFile)
	if err != nil {
		panic(err)
	}

	calSrv, err := getCalendarService(credentialsFile)
	if err != nil {
		panic(err)
	}

	summary := []string{}

	for _, p := range cfg.People {
		if len(weekdays) == 0 {
			break
		}

		fmt.Fprintf(out, "\n%s %s\n", p.FullName, cfg.CtaText)
		for wd, date := range weekdays {
			ok, err := stdInConfirmf(" --> %s:%s", wd.String(), date.Format("2006-01-02"))
			if err != nil {
				panic(err)
			}

			if !ok {
				continue
			}

			event := getEvent(p, date, cfg.StartTime, cfg.EndTime)
			if _, err := calSrv.Events.Insert(cfg.CalID, event).Do(); err != nil {
				panic(err)
			}

			delete(weekdays, wd)
			summary = append(
				summary,
				fmt.Sprintf(
					"--> %s : %s (%s)",
					p.FullName,
					date.Format("2006-01-02"),
					wd.String(),
				))
			break
		}
	}

	fmt.Fprint(out, "\n----------------------\n> assigned weekdays: ", len(summary), "\n----------------------\n")
	for _, item := range summary {
		fmt.Fprintln(out, item)
	}

	if len(weekdays) == 0 {
		return
	}

	fmt.Fprint(out, "\n------------------------\n> unassigned weekdays: ", len(weekdays), "\n------------------------\n")
	for wd, date := range weekdays {
		fmt.Fprintln(out, " -->", wd.String(), ":", date.Format("2006-01-02"))
	}
}

func getToken(cfg *oauth2.Config) (*oauth2.Token, error) {
	tkn, err := tokenFromCache()
	if err == nil {
		return tkn, nil
	}

	authURL := cfg.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Fprintf(out, "> visit the link: %v\n", authURL)

	code, err := stdIn("> enter auth. code:")
	if err != nil {
		return nil, err
	}

	tkn, err = cfg.Exchange(context.Background(), code)
	if err != nil {
		return nil, err
	}

	if err := cacheToken(tkn); err != nil {
		return nil, err
	}

	return tkn, nil
}

func tokenFromCache() (*oauth2.Token, error) {
	f, err := os.Open(tokenCacheFile)
	if err != nil {
		return nil, err
	}

	token := oauth2.Token{}
	if err = json.NewDecoder(f).Decode(&token); err != nil {
		return nil, err
	}
	return &token, nil
}

func cacheToken(tkn *oauth2.Token) error {
	if err := os.MkdirAll(tokenCacheDir, os.ModePerm); err != nil {
		return err
	}

	cacheFile, err := os.Create(tokenCacheFile)
	if err != nil {
		return err
	}

	return json.NewEncoder(cacheFile).Encode(tkn)
}

func getConfig(configFile string) (*Config, error) {
	b, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	var cfg Config
	return &cfg, json.Unmarshal(b, &cfg)
}

func getWeekdays() (map[time.Weekday]time.Time, error) {
	in, err := stdIn("> enter week start date {YYYY-mm-dd}:")
	if err != nil {
		return nil, err
	}

	sd, err := time.Parse("2006-01-02", in)
	if err != nil {
		return nil, err
	}

	if sd.Weekday() < time.Monday || sd.Weekday() > time.Friday {
		fmt.Fprintln(os.Stdout, "> events are not published on the Weekend. Exit.")
		return nil, nil
	}

	y, w := sd.ISOWeek()
	ok, err := stdInConfirmf("> calendar will be filled for week: %d-%d from %s. Proceed?", y, w, sd.Weekday().String())
	if err != nil {
		return nil, err
	}

	if !ok {
		return nil, nil
	}

	weekdays := map[time.Weekday]time.Time{}
	for wd := sd.Weekday(); wd < time.Saturday; wd++ {
		weekdays[wd] = sd
		sd = sd.Add(time.Hour * 24)
	}

	return weekdays, nil
}

func getCalendarService(credentialsFile string) (*calendar.Service, error) {
	b, err := ioutil.ReadFile(credentialsFile)
	if err != nil {
		return nil, err
	}

	cfg, err := google.ConfigFromJSON(b, calendar.CalendarScope)
	if err != nil {
		return nil, err
	}

	tok, err := getToken(cfg)
	if err != nil {
		return nil, err
	}

	return calendar.New(cfg.Client(context.Background(), tok))
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
			&calendar.EventAttendee{Email: p.Email, ResponseStatus: "accepted"},
		},
		Transparency: "transparent",
	}
}

func stdInf(format string, args ...interface{}) {
	txt := fmt.Sprintf(format, args...)
	stdIn(txt)
}

func stdIn(txt string) (string, error) {
	fmt.Fprint(os.Stdout, txt+" ")
	var in string
	_, err := fmt.Fscanln(os.Stdin, &in)
	return in, err
}

func stdInConfirmf(format string, args ...interface{}) (bool, error) {
	txt := fmt.Sprintf(format, args...)
	return stdInConfirm(txt)
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
