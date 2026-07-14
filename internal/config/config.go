// Package config loads and represents the platform configuration file.
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config is the top-level configuration for the drift detector.
type Config struct {
	AWS      AWSConfig      `yaml:"aws"`
	State    StateConfig    `yaml:"state"`
	Drift    DriftConfig    `yaml:"drift"`
	Schedule ScheduleConfig `yaml:"schedule"`
	Storage  StorageConfig  `yaml:"storage"`
	Web      WebConfig      `yaml:"web"`
}

// AWSConfig holds cloud credentials/region settings.
type AWSConfig struct {
	Region  string `yaml:"region"`
	Profile string `yaml:"profile"`
	// AccessKeyID / SecretAccessKey are optional inline credentials.
	AccessKeyID     string `yaml:"access_key_id"`
	SecretAccessKey string `yaml:"secret_access_key"`
}

// StateConfig describes where the Terraform state comes from.
// Source is a local path or an s3://bucket/key URL.
type StateConfig struct {
	Source string `yaml:"source"`
}

// DriftConfig toggles which kinds of drift are evaluated.
type DriftConfig struct {
	CompareAttributes bool `yaml:"compare_attributes"`
	CompareTags       bool `yaml:"compare_tags"`
	DetectOrphans     bool `yaml:"detect_orphans"`
}

// ScheduleConfig controls the recurring scan cadence (cron expression).
type ScheduleConfig struct {
	Spec string `yaml:"spec"`
}

// StorageConfig points at the SQLite database file.
type StorageConfig struct {
	Path string `yaml:"path"`
}

// WebConfig configures the dashboard HTTP server.
type WebConfig struct {
	Addr string `yaml:"addr"`
}

// Default returns a configuration populated with sensible defaults.
func Default() *Config {
	return &Config{
		AWS:      AWSConfig{Region: "us-east-1"},
		State:    StateConfig{Source: "./terraform.tfstate"},
		Drift:    DriftConfig{CompareAttributes: true, CompareTags: true, DetectOrphans: false},
		Schedule: ScheduleConfig{Spec: ""},
		Storage:  StorageConfig{Path: "./driftdetect.db"},
		Web:      WebConfig{Addr: ":8080"},
	}
}

// Load reads and parses the YAML configuration file at path. If path is empty,
// the default configuration is returned. Values not present in the file fall
// back to the defaults defined in Default().
func Load(path string) (*Config, error) {
	cfg := Default()
	if path == "" {
		return cfg, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config %q: %w", path, err)
	}
	return cfg, nil
}
