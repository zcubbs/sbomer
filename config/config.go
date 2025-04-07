package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Database DatabaseConfig `mapstructure:"database"`
	GitLab   GitLabConfig   `mapstructure:"gitlab"`
	AMQP     AMQPConfig     `mapstructure:"amqp"`
	Syft     SyftConfig     `mapstructure:"syft"`
	Fetcher  FetcherConfig  `mapstructure:"fetcher"`
}

type AppConfig struct {
	LogLevel string `mapstructure:"log_level"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

type GitLabConfig struct {
	Host    string `mapstructure:"host"`
	Scheme  string `mapstructure:"scheme"`
	Token   string `mapstructure:"token"`
	TempDir string `mapstructure:"temp_dir"`
}

type AMQPConfig struct {
	URI           string `mapstructure:"uri"`
	Exchange      string `mapstructure:"exchange"`
	ExchangeType  string `mapstructure:"exchange_type"`
	RoutingKey    string `mapstructure:"routing_key"`
	ConsumerGroup string `mapstructure:"consumer_group"`
}

type SyftConfig struct {
	Format      string `mapstructure:"format"`
	SyftBinPath string `mapstructure:"syft_bin_path"`
}

type FetcherConfig struct {
	Schedule      string   `mapstructure:"schedule"`
	BatchSize     int      `mapstructure:"batch_size"`
	CoolOffSecs   int      `mapstructure:"cool_off_secs"`
	GroupIDs      []string `mapstructure:"group_ids"`
	ExcludeTopics []string `mapstructure:"exclude_topics"`
	IncludeTopics []string `mapstructure:"include_topics"`
}

func (c *Config) GetDatabaseURI() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.Database.User,
		c.Database.Password,
		c.Database.Host,
		c.Database.Port,
		c.Database.DBName,
		c.Database.SSLMode,
	)
}

func LoadConfig(configPath string) (*Config, error) {
	var config Config

	// Load .env file if it exists
	if err := loadEnvFile(); err != nil {
		return nil, fmt.Errorf("error loading .env file: %w", err)
	}

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	// Add config path
	if configPath != "" {
		viper.AddConfigPath(configPath)
	}

	// Add default config paths
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/etc/sbomer")

	// Default values
	defaultConfig := Config{
		App: AppConfig{
			LogLevel: "info",
		},
		GitLab: GitLabConfig{
			Host:    "gitlab.com",
			Scheme:  "https",
			TempDir: "tmp",
		},
		AMQP: AMQPConfig{
			URI:           "amqp://guest:guest@localhost:5672/",
			Exchange:      "sbomer",
			ExchangeType:  "fanout",
			RoutingKey:    "",
			ConsumerGroup: "sbomer-group",
		},
		Fetcher: FetcherConfig{
			Schedule:      "once", // Run once for development
			BatchSize:     10,
			CoolOffSecs:   5,
			GroupIDs:      []string{}, // Empty by default, will fetch all projects if not specified
			ExcludeTopics: []string{}, // Empty by default, no topics excluded
		},
		Syft: SyftConfig{
			Format:      "cyclonedx-json",
			SyftBinPath: "syft",
		},
	}

	// Set default values
	viper.SetDefault("app.log_level", defaultConfig.App.LogLevel)
	viper.SetDefault("database.sslmode", "disable")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("gitlab.host", defaultConfig.GitLab.Host)
	viper.SetDefault("gitlab.scheme", defaultConfig.GitLab.Scheme)
	viper.SetDefault("gitlab.temp_dir", defaultConfig.GitLab.TempDir)
	viper.SetDefault("amqp.uri", defaultConfig.AMQP.URI)
	viper.SetDefault("amqp.exchange", defaultConfig.AMQP.Exchange)
	viper.SetDefault("amqp.exchange_type", defaultConfig.AMQP.ExchangeType)
	viper.SetDefault("amqp.routing_key", defaultConfig.AMQP.RoutingKey)
	viper.SetDefault("amqp.consumer_group", defaultConfig.AMQP.ConsumerGroup)
	viper.SetDefault("syft.format", defaultConfig.Syft.Format)
	viper.SetDefault("syft.syft_bin_path", defaultConfig.Syft.SyftBinPath)
	viper.SetDefault("fetcher.schedule", defaultConfig.Fetcher.Schedule)
	viper.SetDefault("fetcher.batch_size", defaultConfig.Fetcher.BatchSize)
	viper.SetDefault("fetcher.cool_off_secs", defaultConfig.Fetcher.CoolOffSecs)
	viper.SetDefault("fetcher.exclude_topics", defaultConfig.Fetcher.ExcludeTopics)
	viper.SetDefault("fetcher.include_topics", defaultConfig.Fetcher.IncludeTopics)

	// Read environment variables
	viper.AutomaticEnv()
	viper.SetEnvPrefix("SBOMER")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Environment variables
	viper.BindEnv("app.log_level", "SBOMER_LOG_LEVEL")
	viper.BindEnv("database.host", "SBOMER_DB_HOST")
	viper.BindEnv("database.port", "SBOMER_DB_PORT")
	viper.BindEnv("database.user", "SBOMER_DB_USER")
	viper.BindEnv("database.password", "SBOMER_DB_PASSWORD")
	viper.BindEnv("database.dbname", "SBOMER_DB_NAME")
	viper.BindEnv("database.sslmode", "SBOMER_DB_SSLMODE")
	viper.BindEnv("gitlab.host", "SBOMER_GITLAB_HOST")
	viper.BindEnv("gitlab.scheme", "SBOMER_GITLAB_SCHEME")
	viper.BindEnv("gitlab.token", "SBOMER_GITLAB_TOKEN")
	viper.BindEnv("gitlab.temp_dir", "SBOMER_GITLAB_TEMP_DIR")
	viper.BindEnv("amqp.uri", "SBOMER_AMQP_URI")
	viper.BindEnv("amqp.exchange", "SBOMER_AMQP_EXCHANGE")
	viper.BindEnv("amqp.exchange_type", "SBOMER_AMQP_EXCHANGE_TYPE")
	viper.BindEnv("amqp.routing_key", "SBOMER_AMQP_ROUTING_KEY")
	viper.BindEnv("amqp.consumer_group", "SBOMER_AMQP_CONSUMER_GROUP")
	viper.BindEnv("syft.format", "SBOMER_SYFT_FORMAT")
	viper.BindEnv("syft.syft_bin_path", "SBOMER_SYFT_BIN_PATH")
	viper.BindEnv("fetcher.schedule", "SBOMER_FETCHER_SCHEDULE")
	viper.BindEnv("fetcher.batch_size", "SBOMER_FETCHER_BATCH_SIZE")
	viper.BindEnv("fetcher.cool_off_secs", "SBOMER_FETCHER_COOL_OFF_SECS")
	viper.BindEnv("fetcher.group_ids", "SBOMER_FETCHER_GROUP_IDS")
	viper.BindEnv("fetcher.exclude_topics", "SBOMER_FETCHER_EXCLUDE_TOPICS")
	viper.BindEnv("fetcher.include_topics", "SBOMER_FETCHER_INCLUDE_TOPICS")

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Unmarshal config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	return &config, nil
}

func loadEnvFile() error {
	envFile := ".env"

	// Check if .env file exists in current directory
	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		// If not found, try to find it in parent directories
		dir, err := os.Getwd()
		if err != nil {
			return err
		}

		for {
			envPath := filepath.Join(dir, envFile)
			if _, err := os.Stat(envPath); err == nil {
				envFile = envPath
				break
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				// Reached root directory
				return nil
			}
			dir = parent
		}
	}

	return godotenv.Load(envFile)
}
