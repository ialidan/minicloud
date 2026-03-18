// Package config handles loading and merging configuration from
// defaults, YAML file, and environment variables (in that precedence order).
package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Duration wraps time.Duration to support YAML unmarshaling from
// both Go duration strings ("30s", "5m") and integer seconds (30).
type Duration time.Duration

func (d Duration) Std() time.Duration { return time.Duration(d) }

func (d *Duration) UnmarshalYAML(node *yaml.Node) error {
	// Use the YAML node tag to distinguish types reliably.
	// Bare integers (tag !!int) are treated as seconds.
	if node.Tag == "!!int" || node.Tag == "!!float" {
		var secs int64
		if err := node.Decode(&secs); err != nil {
			return fmt.Errorf("cannot parse duration seconds: %w", err)
		}
		*d = Duration(time.Duration(secs) * time.Second)
		return nil
	}

	// Strings are parsed as Go duration format ("30s", "2m30s").
	var s string
	if err := node.Decode(&s); err != nil {
		return fmt.Errorf("cannot parse duration: %w", err)
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}
	*d = Duration(parsed)
	return nil
}

// ---------------------------------------------------------------------------
// Config types
// ---------------------------------------------------------------------------

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Storage  StorageConfig  `yaml:"storage"`
	Database DatabaseConfig `yaml:"database"`
	Log      LogConfig      `yaml:"log"`
}

type ServerConfig struct {
	Host              string    `yaml:"host"`
	Port              int       `yaml:"port"`
	ReadTimeout       Duration  `yaml:"read_timeout"`
	WriteTimeout      Duration  `yaml:"write_timeout"`
	IdleTimeout       Duration  `yaml:"idle_timeout"`
	ReadHeaderTimeout Duration  `yaml:"read_header_timeout"`
	MaxUploadSize     int64     `yaml:"max_upload_size"` // bytes
	SecureCookies     bool      `yaml:"secure_cookies"`  // set true behind HTTPS
	TLS               TLSConfig `yaml:"tls"`
}

type TLSConfig struct {
	Enabled  bool   `yaml:"enabled"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}

type StorageConfig struct {
	DataDir string `yaml:"data_dir"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"` // defaults to <data_dir>/minicloud.db
}

type LogConfig struct {
	Level  string `yaml:"level"`  // debug | info | warn | error
	Format string `yaml:"format"` // json | text
}

// SlogLevel maps the config string to slog.Level.
func (l LogConfig) SlogLevel() slog.Level {
	switch strings.ToLower(l.Level) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// ---------------------------------------------------------------------------
// Defaults / Load / Merge
// ---------------------------------------------------------------------------

// Defaults returns a Config with sane production defaults.
func Defaults() *Config {
	return &Config{
		Server: ServerConfig{
			Host:              "0.0.0.0",
			Port:              8080,
			ReadTimeout:       Duration(30 * time.Second),
			WriteTimeout:      Duration(60 * time.Second),
			IdleTimeout:       Duration(120 * time.Second),
			ReadHeaderTimeout: Duration(10 * time.Second),
			MaxUploadSize:     100 << 20, // 100 MiB
		},
		Storage: StorageConfig{
			DataDir: "./data",
		},
		Log: LogConfig{
			Level:  "info",
			Format: "json",
		},
	}
}

// Load reads configuration from a YAML file. If path is empty the loader
// looks for "minicloud.yaml" in the current directory; a missing default
// file is silently ignored. An explicitly provided path that does not exist
// returns an error.
func Load(path string) (*Config, error) {
	cfg := Defaults()

	isDefault := path == ""
	if isDefault {
		path = "minicloud.yaml"
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) && isDefault {
			// No default config file — perfectly fine, use defaults.
			return cfg, nil
		}
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}

	return cfg, nil
}

// ApplyEnv overrides config fields from MINICLOUD_* environment variables.
// Env vars have the highest precedence (above config file values).
func (c *Config) ApplyEnv() {
	if v := os.Getenv("MINICLOUD_HOST"); v != "" {
		c.Server.Host = v
	}
	if v := os.Getenv("MINICLOUD_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			c.Server.Port = port
		}
	}
	if v := os.Getenv("MINICLOUD_DATA_DIR"); v != "" {
		c.Storage.DataDir = v
	}
	if v := os.Getenv("MINICLOUD_DB_PATH"); v != "" {
		c.Database.Path = v
	}
	if v := os.Getenv("MINICLOUD_LOG_LEVEL"); v != "" {
		c.Log.Level = strings.ToLower(v)
	}
	if v := os.Getenv("MINICLOUD_LOG_FORMAT"); v != "" {
		c.Log.Format = strings.ToLower(v)
	}
	if v := os.Getenv("MINICLOUD_MAX_UPLOAD_SIZE"); v != "" {
		if size, err := strconv.ParseInt(v, 10, 64); err == nil {
			c.Server.MaxUploadSize = size
		}
	}
	if v := os.Getenv("MINICLOUD_TLS_ENABLED"); v != "" {
		c.Server.TLS.Enabled = strings.EqualFold(v, "true") || v == "1"
	}
	if v := os.Getenv("MINICLOUD_TLS_CERT"); v != "" {
		c.Server.TLS.CertFile = v
	}
	if v := os.Getenv("MINICLOUD_TLS_KEY"); v != "" {
		c.Server.TLS.KeyFile = v
	}
	if v := os.Getenv("MINICLOUD_SECURE_COOKIES"); v != "" {
		c.Server.SecureCookies = strings.EqualFold(v, "true") || v == "1"
	}
}

// Addr returns the listen address as "host:port".
func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

// DBPath returns the SQLite database file path.
// Defaults to <data_dir>/minicloud.db when not explicitly configured.
func (c *Config) DBPath() string {
	if c.Database.Path != "" {
		return c.Database.Path
	}
	return c.Storage.DataDir + "/minicloud.db"
}
