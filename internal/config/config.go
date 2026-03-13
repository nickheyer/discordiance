package config

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server" json:"server"`
	Database DatabaseConfig `mapstructure:"database" json:"database"`
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

func Load(configPath string) (*Config, error) {
	slog.Info("config: loading configuration", "config_path", configPath)

	v := viper.New()

	v.SetConfigName("config")
	v.SetConfigType("yaml")

	if configPath != "" {
		v.AddConfigPath(configPath)
		slog.Info("config: added custom config path", "path", configPath)
	}
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	v.AddConfigPath("/etc/discordiance")

	setDefaults(v)
	slog.Info("config: defaults set")

	v.SetEnvPrefix("DISCORDIANCE")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	slog.Info("config: environment variables enabled", "prefix", "DISCORDIANCE")

	if err := v.ReadInConfig(); err != nil {
		slog.Info("config: no config file found, using defaults and env vars", "error", err)
	} else {
		slog.Info("config: loaded config file", "file", v.ConfigFileUsed())
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	if err := validateConfig(&cfg); err != nil {
		return nil, fmt.Errorf("config validation error: %w", err)
	}

	slog.Info("config: loaded successfully",
		"server_port", cfg.Server.Port,
		"server_host", cfg.Server.Host,
		"database_path", cfg.Database.Path)

	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("server.port", "8067")
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.read_timeout", 15)
	v.SetDefault("server.write_timeout", 15)
	v.SetDefault("server.idle_timeout", 60)

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
