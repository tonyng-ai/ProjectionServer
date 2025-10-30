package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the master YAML configuration
type Config struct {
	Source   DatabaseConfig `yaml:"source"`
	Target   DatabaseConfig `yaml:"target"`
	Defaults DefaultConfig  `yaml:"defaults"`
	Tables   []TableConfig  `yaml:"tables"`
	API      APIConfig      `yaml:"api"`
}

// DatabaseConfig represents database connection configuration
type DatabaseConfig struct {
	Type     string `yaml:"type"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Database string `yaml:"database"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	SSLMode  string `yaml:"sslmode,omitempty"`
}

// DefaultConfig represents default sync configuration
type DefaultConfig struct {
	RefreshRate        int  `yaml:"refresh_rate"`
	ProtoActorTrigger  bool `yaml:"proto_actor_trigger"`
	WebAPITrigger      bool `yaml:"webapi_trigger"`
	CreateTargetTable  bool `yaml:"create_target_table"`
}

// TableConfig represents individual table sync configuration
type TableConfig struct {
	SourceTable        string   `yaml:"source_table"`
	TargetTable        string   `yaml:"target_table"`
	SyncAction         string   `yaml:"sync_action"`
	RefreshRate        *int     `yaml:"refresh_rate,omitempty"`
	ProtoActorTrigger  *bool    `yaml:"proto_actor_trigger,omitempty"`
	WebAPITrigger      *bool    `yaml:"webapi_trigger,omitempty"`
	Fields             []string `yaml:"fields,omitempty"`
	Filter             string   `yaml:"filter,omitempty"`
}

// APIConfig represents API server configuration
type APIConfig struct {
	Host       string `yaml:"host"`
	Port       int    `yaml:"port"`
	EnableCORS bool   `yaml:"enable_cors"`
}

// GetRefreshRate returns the refresh rate for this table (or default)
func (tc *TableConfig) GetRefreshRate(defaults DefaultConfig) int {
	if tc.RefreshRate != nil {
		return *tc.RefreshRate
	}
	return defaults.RefreshRate
}

// GetProtoActorTrigger returns whether ProtoActor trigger is enabled
func (tc *TableConfig) GetProtoActorTrigger(defaults DefaultConfig) bool {
	if tc.ProtoActorTrigger != nil {
		return *tc.ProtoActorTrigger
	}
	return defaults.ProtoActorTrigger
}

// GetWebAPITrigger returns whether WebAPI trigger is enabled
func (tc *TableConfig) GetWebAPITrigger(defaults DefaultConfig) bool {
	if tc.WebAPITrigger != nil {
		return *tc.WebAPITrigger
	}
	return defaults.WebAPITrigger
}

// GetConnectionString returns the connection string for the database
func (dc *DatabaseConfig) GetConnectionString() string {
	switch dc.Type {
	case "mssql":
		return fmt.Sprintf("sqlserver://%s:%s@%s:%d?database=%s",
			dc.Username, dc.Password, dc.Host, dc.Port, dc.Database)
	case "postgresql":
		sslmode := dc.SSLMode
		if sslmode == "" {
			sslmode = "disable"
		}
		return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
			dc.Username, dc.Password, dc.Host, dc.Port, dc.Database, sslmode)
	default:
		return ""
	}
}

// LoadConfig loads configuration from YAML file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}
