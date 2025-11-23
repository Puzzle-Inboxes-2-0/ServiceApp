package config

import (
	"bytes"
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

const (
	configFileName = "config"
	configFileType = "yaml"
	configFilePath = "."
)

// Config holds the application configuration
type Config struct {
	Environment string           `mapstructure:"environment"`
	Server      ServerConfig     `mapstructure:"server"`
	Database    DatabaseConfig   `mapstructure:"database"`
	Logger      LoggerConfig     `mapstructure:"logging"`
	Monitoring  MonitoringConfig `mapstructure:"monitoring"`
	Ionos       IonosConfig      `mapstructure:"ionos"`
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port         string        `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            string        `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	Name            string        `mapstructure:"name"`
	SSLMode         string        `mapstructure:"sslmode"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

// LoggerConfig holds logging configuration
type LoggerConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// MonitoringConfig holds monitoring configuration
type MonitoringConfig struct {
	Enabled    bool             `mapstructure:"enabled"`
	Prometheus PrometheusConfig `mapstructure:"prometheus"`
}

// PrometheusConfig holds Prometheus-specific configuration
type PrometheusConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Path    string `mapstructure:"path"`
}

// IonosConfig holds IONOS API configuration
type IonosConfig struct {
	Token                  string        `mapstructure:"token"`
	APIURL                 string        `mapstructure:"api_url"`
	DefaultLocation        string        `mapstructure:"default_location"`
	DefaultReservationSize int           `mapstructure:"default_reservation_size"`
	MaxQuota               int           `mapstructure:"max_quota"`
	ReservationTimeout     time.Duration `mapstructure:"reservation_timeout"`
}

// Load reads and parses the configuration file
func Load() (*Config, error) {
	// Set config file details
	viper.SetConfigName(configFileName)
	viper.SetConfigType(configFileType)
	viper.AddConfigPath(configFilePath)

	// Read the config file
	configContent, err := os.ReadFile(fmt.Sprintf("%s/%s.%s", configFilePath, configFileName, configFileType))
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Evaluate environment variables in config
	evaluatedConfig := evaluateString(string(configContent))

	// Read config from the evaluated string
	viper.SetConfigType(configFileType)
	if err := viper.ReadConfig(bytes.NewBuffer([]byte(evaluatedConfig))); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Unmarshal into Config struct
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

// GetDatabaseDSN returns the PostgreSQL connection string
func (c *Config) GetDatabaseDSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.Name,
		c.Database.SSLMode,
	)
}
