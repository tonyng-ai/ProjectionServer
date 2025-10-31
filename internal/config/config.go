package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the master YAML configuration
type Config struct {
	Source      DatabaseConfig     `yaml:"source"`
	Target      DatabaseConfig     `yaml:"target"`
	Defaults    DefaultConfig      `yaml:"defaults"`
	Tables      []TableConfig      `yaml:"tables"`
	API         APIConfig          `yaml:"api"`
	Projections []ProjectionConfig `yaml:"projections"`
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
	RefreshRate       int  `yaml:"refresh_rate"`
	ProtoActorTrigger bool `yaml:"proto_actor_trigger"`
	WebAPITrigger     bool `yaml:"webapi_trigger"`
	CreateTargetTable bool `yaml:"create_target_table"`
}

// TableConfig represents individual table sync configuration
type TableConfig struct {
	SourceTable       string   `yaml:"source_table"`
	TargetTable       string   `yaml:"target_table"`
	SyncAction        string   `yaml:"sync_action"`
	RefreshRate       *int     `yaml:"refresh_rate,omitempty"`
	ProtoActorTrigger *bool    `yaml:"proto_actor_trigger,omitempty"`
	WebAPITrigger     *bool    `yaml:"webapi_trigger,omitempty"`
	Fields            []string `yaml:"fields,omitempty"`
	Filter            string   `yaml:"filter,omitempty"`
}

// ProjectionConfig represents UI projection configuration for a target view
type ProjectionConfig struct {
	ID              string                   `yaml:"id" json:"id"`
	Title           string                   `yaml:"title" json:"title"`
	Description     string                   `yaml:"description,omitempty" json:"description,omitempty"`
	TargetView      string                   `yaml:"target_view" json:"target_view"`
	SyncTable       string                   `yaml:"sync_table" json:"sync_table"`
	HeaderColor     string                   `yaml:"header_color,omitempty" json:"header_color,omitempty"`
	HeaderTextColor string                   `yaml:"header_text_color,omitempty" json:"header_text_color,omitempty"`
	DefaultSort     *ProjectionSortConfig    `yaml:"default_sort,omitempty" json:"default_sort,omitempty"`
	GroupBy         []string                 `yaml:"group_by,omitempty" json:"group_by,omitempty"`
	Fields          []ProjectionFieldConfig  `yaml:"fields,omitempty" json:"fields,omitempty"`
	Filters         []ProjectionFilterConfig `yaml:"filters,omitempty" json:"filters,omitempty"`
	Totals          []ProjectionTotalConfig  `yaml:"totals,omitempty" json:"totals,omitempty"`
}

// ProjectionFieldConfig describes a field to display in the UI
type ProjectionFieldConfig struct {
	Column   string `yaml:"column" json:"column"`
	Label    string `yaml:"label" json:"label"`
	Type     string `yaml:"type,omitempty" json:"type,omitempty"`
	Sortable *bool  `yaml:"sortable,omitempty" json:"sortable,omitempty"`
}

// ProjectionFilterConfig describes a filter input for the UI
type ProjectionFilterConfig struct {
	ID      string                         `yaml:"id" json:"id"`
	Column  string                         `yaml:"column" json:"column"`
	Label   string                         `yaml:"label" json:"label"`
	Type    string                         `yaml:"type" json:"type"`
	Options []ProjectionFilterOptionConfig `yaml:"options,omitempty" json:"options,omitempty"`
}

// ProjectionFilterOptionConfig describes a selectable filter option
type ProjectionFilterOptionConfig struct {
	Label string `yaml:"label" json:"label"`
	Value string `yaml:"value" json:"value"`
}

// ProjectionSortConfig describes default sorting
type ProjectionSortConfig struct {
	Column    string `yaml:"column" json:"column"`
	Direction string `yaml:"direction" json:"direction"`
}

// ProjectionTotalConfig describes a total aggregation for a column
type ProjectionTotalConfig struct {
	Column string `yaml:"column" json:"column"`
	Label  string `yaml:"label" json:"label"`
	Format string `yaml:"format,omitempty" json:"format,omitempty"`
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

// GetProjectionByID returns a projection configuration by its identifier
func (c *Config) GetProjectionByID(id string) (*ProjectionConfig, bool) {
	for i := range c.Projections {
		if c.Projections[i].ID == id {
			return &c.Projections[i], true
		}
	}
	return nil, false
}
