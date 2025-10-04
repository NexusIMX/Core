package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Consul   ConsulConfig   `mapstructure:"consul"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	S3       S3Config       `mapstructure:"s3"`
	Log      LogConfig      `mapstructure:"log"`
	Message  MessageConfig  `mapstructure:"message"`
	File     FileConfig     `mapstructure:"file"`
}

type ServerConfig struct {
	Gateway GatewayConfig    `mapstructure:"gateway"`
	Router  RouterConfig     `mapstructure:"router"`
	Message MessageSvcConfig `mapstructure:"message"`
	User    UserConfig       `mapstructure:"user"`
	File    FileSvcConfig    `mapstructure:"file"`
}

type ConsulConfig struct {
	Address             string        `mapstructure:"address"`
	Scheme              string        `mapstructure:"scheme"`
	Datacenter          string        `mapstructure:"datacenter"`
	HealthCheckInterval time.Duration `mapstructure:"health_check_interval"`
	DeregisterAfter     time.Duration `mapstructure:"deregister_after"`
}

type GatewayConfig struct {
	GRPCPort int `mapstructure:"grpc_port"`
}

type RouterConfig struct {
	GRPCPort int `mapstructure:"grpc_port"`
}

type MessageSvcConfig struct {
	GRPCPort int `mapstructure:"grpc_port"`
}

type UserConfig struct {
	GRPCPort int `mapstructure:"grpc_port"`
}

type FileSvcConfig struct {
	HTTPPort int `mapstructure:"http_port"`
}

type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	DBName          string        `mapstructure:"dbname"`
	SSLMode         string        `mapstructure:"sslmode"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	PoolSize int    `mapstructure:"pool_size"`
}

type JWTConfig struct {
	Secret string        `mapstructure:"secret"`
	Expiry time.Duration `mapstructure:"expiry"`
}

type S3Config struct {
	Endpoint  string `mapstructure:"endpoint"`
	Region    string `mapstructure:"region"`
	Bucket    string `mapstructure:"bucket"`
	AccessKey string `mapstructure:"access_key"`
	SecretKey string `mapstructure:"secret_key"`
	UseSSL    bool   `mapstructure:"use_ssl"`
}

type LogConfig struct {
	Level       string   `mapstructure:"level"`
	Encoding    string   `mapstructure:"encoding"`
	OutputPaths []string `mapstructure:"output_paths"`
}

type MessageConfig struct {
	RetentionDays int `mapstructure:"retention_days"`
	MaxPullLimit  int `mapstructure:"max_pull_limit"`
}

type FileConfig struct {
	MaxSizeMB    int      `mapstructure:"max_size_mb"`
	AllowedTypes []string `mapstructure:"allowed_types"`
}

// Load loads configuration from file and environment variables
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set config file
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	// Read from environment variables
	v.AutomaticEnv()

	// Environment variable overrides
	v.BindEnv("database.host", "POSTGRES_HOST")
	v.BindEnv("database.port", "POSTGRES_PORT")
	v.BindEnv("database.user", "POSTGRES_USER")
	v.BindEnv("database.password", "POSTGRES_PASSWORD")
	v.BindEnv("database.dbname", "POSTGRES_DB")

	v.BindEnv("redis.host", "REDIS_HOST")
	v.BindEnv("redis.port", "REDIS_PORT")
	v.BindEnv("redis.password", "REDIS_PASSWORD")

	v.BindEnv("jwt.secret", "JWT_SECRET")
	v.BindEnv("jwt.expiry", "JWT_EXPIRY")

	v.BindEnv("s3.endpoint", "S3_ENDPOINT")
	v.BindEnv("s3.region", "S3_REGION")
	v.BindEnv("s3.bucket", "S3_BUCKET")
	v.BindEnv("s3.access_key", "S3_ACCESS_KEY")
	v.BindEnv("s3.secret_key", "S3_SECRET_KEY")
	v.BindEnv("s3.use_ssl", "S3_USE_SSL")

	v.BindEnv("server.gateway.grpc_port", "GATEWAY_GRPC_PORT")
	v.BindEnv("server.router.grpc_port", "ROUTER_GRPC_PORT")
	v.BindEnv("server.message.grpc_port", "MESSAGE_GRPC_PORT")
	v.BindEnv("server.user.grpc_port", "USER_GRPC_PORT")
	v.BindEnv("server.file.http_port", "FILE_HTTP_PORT")

	v.BindEnv("log.level", "LOG_LEVEL")

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}
