package config

import (
	"encoding/json"
	"os"
	"time"

	"github.com/makarski/gcaler/staff"
)

type (
	// Config struct describes the app config
	Config struct {
		Templates []Template
	}

	// Template holds calendar event basic configuration data
	Template struct {
		CalID        string           `json:"cal_id"`
		Name         string           `json:"name"`
		EventName    string           `json:"event_name"`
		StartTimeTZ  string           `json:"start_time_tz"`
		Participants []staff.Assignee `json:"participants"`
		DurationStr  string           `json:"duration"`
		Duration     time.Duration    `json:"-"`
	}
)

// UnmarshalJSON is implemented to parse string event duration to time.Duration
func (t *Template) UnmarshalJSON(b []byte) error {
	type Alias Template

	var raw Alias
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}

	d, err := time.ParseDuration(raw.DurationStr)
	if err != nil {
		return err
	}

	*t = Template(raw)
	t.Duration = d

	return nil
}

// Load reads config file from the disk
// and returns a deserialized struct on success
func Load(file string) (*Config, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cfg Config
	err = json.NewDecoder(f).Decode(&cfg)

	return &cfg, err
}
