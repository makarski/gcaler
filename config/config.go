package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pelletier/go-toml"
)

const (
	RecModeSingle    RecMode = "single"
	RecModeRecurrent RecMode = "recurrent"
)

type (
	// Config struct describes the app config
	Config struct {
		Templates []Template
	}

	// Template holds calendar event basic configuration data
	Template struct {
		CalID                 string        `toml:"cal_id"`
		Name                  string        `toml:"name"`
		EventTitle            string        `toml:"event_title"`
		Timezone              string        `toml:"timezone"`
		Transparency          string        `toml:"transparency"`
		Visibility            string        `toml:"visibility"`
		Participants          []*Assignee   `toml:"participants"`
		EventHost             Assignee      `toml:"host"`
		Duration              time.Duration `toml:"duration"`
		Recurrence            Recurrence    `toml:"recurrence"`
		Description           string        `toml:"description"`
		TitleWithParticipants bool          `toml:"title_with_participants"`
	}

	// Assignee describes a config `people` item entry
	Assignee struct {
		FirstName   string `toml:"first_name"`
		LastName    string `toml:"last_name"`
		Email       string `toml:"email"`
		Description string `toml:"description"`
	}

	Recurrence struct {
		Mode      RecMode       `toml:"mode"`
		Count     uint32        `toml:"count"`
		Frequency time.Duration `toml:"frequency"`
		Interval  uint32        `toml:"interval"`
	}

	RecMode string
)

func LoadTemplate(file string) (*Template, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cfg Template
	if err := toml.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, err
	}

	cfg.applyDescriptions()

	return &cfg, cfg.validate()
}

func (t *Template) validate() error {
	validators := []func() error{
		t.Recurrence.validate,
		t.validateTransparency,
		t.validateVisibility,
	}

	errs := make([]string, 0)
	for _, validator := range validators {
		if err := validator(); err != nil {
			errs = append(errs, err.Error())
		}
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}

	return nil
}

func (t *Template) validateVisibility() error {
	validValues := map[string]bool{
		"public":  true,
		"private": true,
		"":        true,
	}

	if _, ok := validValues[t.Visibility]; !ok {
		return fmt.Errorf("invalid config `visibility` value: %s", t.Visibility)
	}

	return nil
}

func (t *Template) validateTransparency() error {
	validValueMapping := map[string]string{
		"busy": "opaque",
		"free": "transparent",
		"":     "",
	}

	targetValue, ok := validValueMapping[t.Transparency]
	if !ok {
		return fmt.Errorf("invalid config `transparency` value: %s", t.Transparency)
	}

	// overwrite (mutate) config values
	t.Transparency = targetValue

	return nil
}

func (t *Template) GenerateEventTitle(participants ...Assignee) string {
	if !t.TitleWithParticipants {
		return t.EventTitle
	}

	names := make([]string, 0, len(participants))
	for _, p := range participants {
		names = append(names, p.FirstName)
	}

	return fmt.Sprintf("%s: %s", t.EventTitle, strings.Join(names, " / "))
}

func (t *Template) applyDescriptions() {
	for _, participant := range t.Participants {
		if participant.Description == "" {
			participant.Description = t.Description
		}
	}
}

func (r RecMode) IsSingle() bool    { return r == RecModeSingle }
func (r RecMode) IsRecurrent() bool { return r == RecModeRecurrent }

func (r *Recurrence) validate() error {
	if r.Mode.IsSingle() || r.Mode.IsRecurrent() {
		return nil
	}

	if _, err := r.frequency(); err != nil {
		return err
	}

	return fmt.Errorf("unsupported recurrence mode: %s", r.Mode)
}

func (r *Recurrence) RFC5545() ([]string, error) {
	f, err := r.frequency()
	if err != nil {
		return nil, err
	}

	return []string{
		fmt.Sprintf(
			"RRULE:FREQ=%s;COUNT=%d;INTERVAL=%d",
			f,
			r.Count,
			r.Interval,
		),
	}, nil
}

func (r *Recurrence) frequency() (string, error) {
	switch {
	case r.Frequency.Minutes() <= 1:
		return "MINUTELY", nil
	case r.Frequency.Hours() <= 1:
		return "HOURLY", nil
	case r.Frequency.Hours() <= 24:
		return "DAILY", nil
	case r.Frequency.Hours() <= 24*7:
		return "WEEKLY", nil
	case r.Frequency.Hours() <= 24*30:
		return "MONTHLY", nil
	case r.Frequency.Hours() <= 24*365:
		return "YEARLY", nil
	}

	return "", fmt.Errorf("frequence is out of range: %v", r.Frequency)
}

func (a *Assignee) FullName() string { return a.FirstName + " " + a.LastName }
