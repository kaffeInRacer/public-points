package config

import (
	"fmt"
	"github.com/spf13/viper"
)

type Config struct {
	Environment   string             `mapstructure:"environment"`
	Server        ServerConfig       `mapstructure:"server"`
	Database      DatabaseConfig     `mapstructure:"database"`
	Redis         RedisConfig        `mapstructure:"redis"`
	Elasticsearch ElasticsearchConfig `mapstructure:"elasticsearch"`
	JWT           JWTConfig          `mapstructure:"jwt"`
	Midtrans      MidtransConfig     `mapstructure:"midtrans"`
	GRPC          GRPCConfig         `mapstructure:"grpc"`
	SMTP          SMTPConfig         `mapstructure:"smtp"`
	RabbitMQ      RabbitMQConfig     `mapstructure:"rabbitmq"`
	Logger        LoggerConfig       `mapstructure:"logger"`
	Workers       WorkersConfig      `mapstructure:"workers"`
}

type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port string `mapstructure:"port"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type ElasticsearchConfig struct {
	URL      string `mapstructure:"url"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

type JWTConfig struct {
	SecretKey string `mapstructure:"secret_key"`
	ExpiryHours int  `mapstructure:"expiry_hours"`
}

type MidtransConfig struct {
	ServerKey    string `mapstructure:"server_key"`
	ClientKey    string `mapstructure:"client_key"`
	Environment  string `mapstructure:"environment"` // sandbox or production
}

type GRPCConfig struct {
	Host string `mapstructure:"host"`
	Port string `mapstructure:"port"`
}

type SMTPConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	From     string `mapstructure:"from"`
	UseTLS   bool   `mapstructure:"use_tls"`
}

type RabbitMQConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	VHost    string `mapstructure:"vhost"`
}

type LoggerConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	Output     string `mapstructure:"output"`
	FilePath   string `mapstructure:"file_path"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
	Compress   bool   `mapstructure:"compress"`
}

type WorkersConfig struct {
	EmailWorkers        int `mapstructure:"email_workers"`
	InvoiceWorkers      int `mapstructure:"invoice_workers"`
	NotificationWorkers int `mapstructure:"notification_workers"`
	AnalyticsWorkers    int `mapstructure:"analytics_workers"`
	MaxRetries          int `mapstructure:"max_retries"`
	RetryDelay          int `mapstructure:"retry_delay"`
}

func LoadConfig() (*Config, error) {
	// Get environment from ENV variable or default to "development"
	env := viper.GetString("ENVIRONMENT")
	if env == "" {
		env = "development"
	}

	// Set config file name based on environment
	configName := "config"
	if env != "production" {
		configName = fmt.Sprintf("config.%s", env)
	}

	viper.SetConfigName(configName)
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("./configs")

	// Set default values
	setDefaults()

	// Read environment variables
	viper.AutomaticEnv()

	// Try to read environment-specific config first
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok && env != "development" {
			// Fallback to default config if environment-specific config not found
			viper.SetConfigName("config")
			if err := viper.ReadInConfig(); err != nil {
				if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
					return nil, err
				}
			}
		} else if !ok {
			return nil, err
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	// Set environment in config
	config.Environment = env

	return &config, nil
}

func setDefaults() {
	// Environment
	viper.SetDefault("environment", "development")

	// Server defaults
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", "12000")

	// Database defaults
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", "5432")
	viper.SetDefault("database.sslmode", "disable")

	// Redis defaults
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", "6379")
	viper.SetDefault("redis.db", 0)

	// Elasticsearch defaults
	viper.SetDefault("elasticsearch.url", "http://localhost:9200")

	// JWT defaults
	viper.SetDefault("jwt.expiry_hours", 24)

	// Midtrans defaults
	viper.SetDefault("midtrans.environment", "sandbox")

	// GRPC defaults
	viper.SetDefault("grpc.host", "0.0.0.0")
	viper.SetDefault("grpc.port", "12001")

	// SMTP defaults
	viper.SetDefault("smtp.host", "localhost")
	viper.SetDefault("smtp.port", 587)
	viper.SetDefault("smtp.use_tls", true)

	// RabbitMQ defaults
	viper.SetDefault("rabbitmq.host", "localhost")
	viper.SetDefault("rabbitmq.port", 5672)
	viper.SetDefault("rabbitmq.username", "guest")
	viper.SetDefault("rabbitmq.password", "guest")
	viper.SetDefault("rabbitmq.vhost", "/")

	// Logger defaults
	viper.SetDefault("logger.level", "info")
	viper.SetDefault("logger.format", "json")
	viper.SetDefault("logger.output", "stdout")
	viper.SetDefault("logger.file_path", "/var/log/online-shop/app.log")
	viper.SetDefault("logger.max_size", 100)
	viper.SetDefault("logger.max_backups", 3)
	viper.SetDefault("logger.max_age", 28)
	viper.SetDefault("logger.compress", true)

	// Workers defaults
	viper.SetDefault("workers.email_workers", 5)
	viper.SetDefault("workers.invoice_workers", 3)
	viper.SetDefault("workers.notification_workers", 3)
	viper.SetDefault("workers.analytics_workers", 2)
	viper.SetDefault("workers.max_retries", 3)
	viper.SetDefault("workers.retry_delay", 5)
}