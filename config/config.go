package config

import (
	"encoding/json"
	"os"

	"github.com/makarski/gcaler/staff"
)

// Config struct describes the app config
type Config struct {
	CalID     string           `json:"cal_id"`
	StartTime string           `json:"start_time"`
	EndTime   string           `json:"end_time"`
	CtaText   string           `json:"cta_text"`
	People    []staff.Assignee `json:"people"`
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
