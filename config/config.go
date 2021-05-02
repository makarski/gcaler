package config

import (
	"os"
	"time"

	"github.com/pelletier/go-toml"

	"github.com/makarski/gcaler/staff"
)

type (
	// Config struct describes the app config
	Config struct {
		Templates []Template
	}

	// Template holds calendar event basic configuration data
	Template struct {
		CalID        string           `toml:"cal_id"`
		Name         string           `toml:"name"`
		EventName    string           `toml:"event_name"`
		StartTimeTZ  string           `toml:"start_time_tz"`
		Participants []staff.Assignee `toml:"participants"`
		Duration     time.Duration    `toml:"duration"`

		// Not yet used
		Recurrence string `toml:"recurrence"`
	}
)

func LoadTemplate(file string) (*Template, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cfg Template
	err = toml.NewDecoder(f).Decode(&cfg)

	return &cfg, err
}
