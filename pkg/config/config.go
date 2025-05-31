package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	Server       ServerConfig       `mapstructure:"server"`
	Database     DatabaseConfig     `mapstructure:"database"`
	Redis        RedisConfig        `mapstructure:"redis"`
	Elasticsearch ElasticsearchConfig `mapstructure:"elasticsearch"`
	JWT          JWTConfig          `mapstructure:"jwt"`
	Midtrans     MidtransConfig     `mapstructure:"midtrans"`
	GRPC         GRPCConfig         `mapstructure:"grpc"`
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

func LoadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	// Set default values
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", "12000")
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", "5432")
	viper.SetDefault("database.sslmode", "disable")
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", "6379")
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("elasticsearch.url", "http://localhost:9200")
	viper.SetDefault("jwt.expiry_hours", 24)
	viper.SetDefault("midtrans.environment", "sandbox")
	viper.SetDefault("grpc.host", "0.0.0.0")
	viper.SetDefault("grpc.port", "12001")

	// Read environment variables
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}