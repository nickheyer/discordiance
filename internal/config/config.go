package config

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig    `mapstructure:"server" json:"server"`
	Database DatabaseConfig  `mapstructure:"database" json:"database"`
	Products []ProductConfig `mapstructure:"products" json:"products"`
}

type ServerConfig struct {
	Port         string `mapstructure:"port" json:"port"`
	Host         string `mapstructure:"host" json:"host"`
	ReadTimeout  int    `mapstructure:"read_timeout" json:"read_timeout"`
	WriteTimeout int    `mapstructure:"write_timeout" json:"write_timeout"`
	IdleTimeout  int    `mapstructure:"idle_timeout" json:"idle_timeout"`
}

type DatabaseConfig struct {
	Path        string `mapstructure:"path" json:"path"`
	AutoMigrate bool   `mapstructure:"auto_migrate" json:"auto_migrate"`
}

type ProductConfig struct {
	Name      string         `mapstructure:"name" json:"name"`
	Platforms []PlatformSpec `mapstructure:"platforms" json:"platforms"`
	Agent     AgentSpec      `mapstructure:"agent" json:"agent"`
	Reporters []ReporterSpec `mapstructure:"reporters" json:"reporters"`
}

type PlatformSpec struct {
	Type     string            `mapstructure:"type" json:"type"`
	Settings map[string]string `mapstructure:"settings" json:"settings"`
}

type AgentSpec struct {
	BaseURL      string `mapstructure:"base_url" json:"base_url"`
	APIKey       string `mapstructure:"api_key" json:"api_key"`
	OrgID        string `mapstructure:"org_id" json:"org_id"`
	Model        string `mapstructure:"model" json:"model"`
	SystemPrompt string `mapstructure:"system_prompt" json:"system_prompt"`
	BatchSize    int    `mapstructure:"batch_size" json:"batch_size"`
	BatchTimeout int    `mapstructure:"batch_timeout_seconds" json:"batch_timeout_seconds"`
}

type ReporterSpec struct {
	Type     string            `mapstructure:"type" json:"type"`
	Settings map[string]string `mapstructure:"settings" json:"settings"`
}

func Load(configPath string) (*Config, error) {
	slog.Info("config: loading configuration", "config_path", configPath)

	v := viper.New()

	// Set config name and type
	v.SetConfigName("config")
	v.SetConfigType("yaml")

	// Add config paths
	if configPath != "" {
		v.AddConfigPath(configPath)
		slog.Info("config: added custom config path", "path", configPath)
	}
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	v.AddConfigPath("/etc/discordiance")

	// Set defaults
	setDefaults(v)
	slog.Info("config: defaults set")

	// Enable environment variables
	v.SetEnvPrefix("DISCORDIANCE")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	slog.Info("config: environment variables enabled", "prefix", "DISCORDIANCE")

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		slog.Info("config: no config file found, using defaults and env vars", "error", err)
	} else {
		slog.Info("config: loaded config file", "file", v.ConfigFileUsed())
	}

	// Unmarshal config with decode hooks for env var flexibility
	var cfg Config
	if err := v.Unmarshal(&cfg, func(dc *mapstructure.DecoderConfig) {
		dc.DecodeHook = mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
			jsonStringToMapHook(),
		)
	}); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Validate and expand paths
	if err := validateConfig(&cfg); err != nil {
		return nil, fmt.Errorf("config validation error: %w", err)
	}

	slog.Info("config: loaded successfully",
		"server_port", cfg.Server.Port,
		"server_host", cfg.Server.Host,
		"database_path", cfg.Database.Path,
		"products", len(cfg.Products))

	for i, p := range cfg.Products {
		slog.Info("config: product", "index", i, "name", p.Name,
			"platforms", len(p.Platforms), "reporters", len(p.Reporters),
			"agent_model", p.Agent.Model)
	}

	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.port", "8067")
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.read_timeout", 15)
	v.SetDefault("server.write_timeout", 15)
	v.SetDefault("server.idle_timeout", 60)

	// Database defaults
	v.SetDefault("database.path", "./data/discordiance.db")
	v.SetDefault("database.auto_migrate", true)
}

func validateConfig(cfg *Config) error {
	var err error
	cfg.Database.Path, err = filepath.Abs(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("invalid database path: %w", err)
	}
	slog.Info("config: database path resolved", "path", cfg.Database.Path)
	return nil
}

// jsonStringToMapHook decodes JSON object strings into map types (useful for env vars).
func jsonStringToMapHook() mapstructure.DecodeHookFuncType {
	return func(from, to reflect.Type, data any) (any, error) {
		if from.Kind() != reflect.String || to.Kind() != reflect.Map {
			return data, nil
		}
		var m any
		if err := json.Unmarshal([]byte(data.(string)), &m); err != nil {
			return data, nil
		}
		return m, nil
	}
}
