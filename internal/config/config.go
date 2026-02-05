package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config is the top-level application configuration.
type Config struct {
	Anthropic     AnthropicConfig     `mapstructure:"anthropic"`
	Monitoring    MonitoringConfig    `mapstructure:"monitoring"`
	AI            AIConfig            `mapstructure:"ai"`
	Safety        SafetyConfig        `mapstructure:"safety"`
	Notifications NotificationConfig  `mapstructure:"notifications"`
	Power         PowerConfig         `mapstructure:"power"`
}

type AnthropicConfig struct {
	APIKey string `mapstructure:"api_key"`
	Model  string `mapstructure:"model"`
}

type MonitoringConfig struct {
	ScanInterval    time.Duration `mapstructure:"scan_interval"`
	CPUThreshold    float64       `mapstructure:"cpu_threshold"`
	MemoryThreshold float64       `mapstructure:"memory_threshold"`
	IOThreshold     int64         `mapstructure:"io_threshold"`
	HistorySize     int           `mapstructure:"history_size"`
}

type AIConfig struct {
	Enabled            bool    `mapstructure:"enabled"`
	ConfidenceThreshold float64 `mapstructure:"confidence_threshold"`
	CacheSize          int     `mapstructure:"cache_size"`
	CacheTTL           time.Duration `mapstructure:"cache_ttl"`
	MaxRequestsPerMin  int     `mapstructure:"max_requests_per_min"`
	Aggressiveness     int     `mapstructure:"aggressiveness"`
}

type SafetyConfig struct {
	ConsentLevel     int      `mapstructure:"consent_level"`
	ProtectedProcs   []string `mapstructure:"protected_processes"`
	NeverTerminate   []string `mapstructure:"never_terminate"`
	TerminateTimeout time.Duration `mapstructure:"terminate_timeout"`
}

type NotificationConfig struct {
	LogFile      string `mapstructure:"log_file"`
	AuditFile    string `mapstructure:"audit_file"`
	Verbose      bool   `mapstructure:"verbose"`
	ColorEnabled bool   `mapstructure:"color_enabled"`
}

type PowerConfig struct {
	CPUWattPerPercent  float64 `mapstructure:"cpu_watt_per_percent"`
	MemoryWattPerMB    float64 `mapstructure:"memory_watt_per_mb"`
	DiskWattPerMBps    float64 `mapstructure:"disk_watt_per_mbps"`
	TrackSavings       bool    `mapstructure:"track_savings"`
}

func setDefaults() {
	viper.SetDefault("anthropic.model", "claude-sonnet-4-5-20250929")

	viper.SetDefault("monitoring.scan_interval", "2s")
	viper.SetDefault("monitoring.cpu_threshold", 80.0)
	viper.SetDefault("monitoring.memory_threshold", 80.0)
	viper.SetDefault("monitoring.io_threshold", 104857600) // 100MB/s
	viper.SetDefault("monitoring.history_size", 100)

	viper.SetDefault("ai.enabled", true)
	viper.SetDefault("ai.confidence_threshold", 0.7)
	viper.SetDefault("ai.cache_size", 500)
	viper.SetDefault("ai.cache_ttl", "30m")
	viper.SetDefault("ai.max_requests_per_min", 30)
	viper.SetDefault("ai.aggressiveness", 5)

	viper.SetDefault("safety.consent_level", 2)
	viper.SetDefault("safety.protected_processes", []string{
		"systemd", "init", "sshd", "dbus-daemon", "NetworkManager",
		"pulseaudio", "pipewire", "Xorg", "Xwayland", "gnome-shell",
		"kwin", "sway", "gdm", "lightdm", "login",
	})
	viper.SetDefault("safety.never_terminate", []string{
		"systemd", "init", "kernel", "kthreadd",
	})
	viper.SetDefault("safety.terminate_timeout", "5s")

	viper.SetDefault("notifications.log_file", "apo.log")
	viper.SetDefault("notifications.audit_file", "apo-audit.log")
	viper.SetDefault("notifications.verbose", false)
	viper.SetDefault("notifications.color_enabled", true)

	viper.SetDefault("power.cpu_watt_per_percent", 0.5)
	viper.SetDefault("power.memory_watt_per_mb", 0.001)
	viper.SetDefault("power.disk_watt_per_mbps", 0.02)
	viper.SetDefault("power.track_savings", true)
}

// Load reads configuration from file, environment, and defaults.
func Load(configPath string) (*Config, error) {
	setDefaults()

	viper.SetConfigType("yaml")
	viper.SetEnvPrefix("APO")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Allow API key from env
	_ = viper.BindEnv("anthropic.api_key", "ANTHROPIC_API_KEY")

	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		// Search in current dir, home dir, /etc
		viper.SetConfigName("config")
		viper.AddConfigPath(".")
		if home, err := os.UserHomeDir(); err == nil {
			viper.AddConfigPath(filepath.Join(home, ".apo"))
		}
		viper.AddConfigPath("/etc/apo")
	}

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("reading config: %w", err)
		}
		// Config file not found is OK — we use defaults
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return &cfg, nil
}

// Global holds the current loaded configuration.
var Global *Config
