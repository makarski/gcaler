package config

import (
	"fmt"
	"os"
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
		CalID        string        `toml:"cal_id"`
		Name         string        `toml:"name"`
		EventName    string        `toml:"event_name"`
		StartTimeTZ  string        `toml:"start_time_tz"`
		Participants []*Assignee   `toml:"participants"`
		Duration     time.Duration `toml:"duration"`
		Recurrence   Recurrence    `toml:"recurrence"`
		Description  string        `toml:"description"`
	}

	// Assignee describes a config `people` item entry
	Assignee struct {
		FullName    string `toml:"full_name"`
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

	return &cfg, cfg.Recurrence.validate()
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
