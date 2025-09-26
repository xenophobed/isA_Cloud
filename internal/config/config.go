package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	App               AppConfig             `mapstructure:"app"`
	Environment       string                `mapstructure:"environment"`
	Debug             bool                  `mapstructure:"debug"`
	Server            ServerConfig          `mapstructure:"server"`
	Services          ServicesConfig        `mapstructure:"services"`
	Database          DatabaseConfig        `mapstructure:"database"`
	Redis             RedisConfig           `mapstructure:"redis"`
	Logging           LoggingConfig         `mapstructure:"logging"`
	Monitoring        MonitoringConfig      `mapstructure:"monitoring"`
	Security          SecurityConfig        `mapstructure:"security"`
	Blockchain        BlockchainConfig      `mapstructure:"blockchain"`  // Re-enabled with chain-agnostic design
	MQTT              MQTTConfig            `mapstructure:"mqtt"`
	DeviceManagement  DeviceManagementConfig `mapstructure:"device_management"`
}

// AppConfig contains application metadata
type AppConfig struct {
	Name    string `mapstructure:"name"`
	Version string `mapstructure:"version"`
}

// ServerConfig contains server configuration
type ServerConfig struct {
	Host     string `mapstructure:"host"`
	HTTPPort int    `mapstructure:"http_port"`
	GRPCPort int    `mapstructure:"grpc_port"`
}

// ServicesConfig contains external services configuration
type ServicesConfig struct {
	UserService  ServiceEndpoint `mapstructure:"user_service"`
	AuthService  ServiceEndpoint `mapstructure:"auth_service"`
	AgentService ServiceEndpoint `mapstructure:"agent_service"`
	ModelService ServiceEndpoint `mapstructure:"model_service"`
	MCPService   ServiceEndpoint `mapstructure:"mcp_service"`
}

// ServiceEndpoint represents a service endpoint configuration
type ServiceEndpoint struct {
	Host     string        `mapstructure:"host"`
	HTTPPort int           `mapstructure:"http_port"`
	GRPCPort int           `mapstructure:"grpc_port"`
	Timeout  time.Duration `mapstructure:"timeout"`
	Retry    RetryConfig   `mapstructure:"retry"`
}

// RetryConfig contains retry configuration
type RetryConfig struct {
	MaxAttempts int           `mapstructure:"max_attempts"`
	Backoff     time.Duration `mapstructure:"backoff"`
}

// DatabaseConfig contains database configuration
type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Database string `mapstructure:"database"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	SSLMode  string `mapstructure:"ssl_mode"`
}

// RedisConfig contains Redis configuration
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	Database int    `mapstructure:"database"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"` // json, text
	Output string `mapstructure:"output"` // stdout, file
	File   string `mapstructure:"file"`
}

// MonitoringConfig contains monitoring configuration
type MonitoringConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	Port       int    `mapstructure:"port"`
	Path       string `mapstructure:"path"`
	Namespace  string `mapstructure:"namespace"`
	Subsystem  string `mapstructure:"subsystem"`
}

// SecurityConfig contains security configuration
type SecurityConfig struct {
	CORS        CORSConfig `mapstructure:"cors"`
	RateLimit   RateLimitConfig `mapstructure:"rate_limit"`
	JWT         JWTConfig `mapstructure:"jwt"`
}

// CORSConfig contains CORS configuration
type CORSConfig struct {
	Enabled          bool     `mapstructure:"enabled"`
	AllowOrigins     []string `mapstructure:"allow_origins"`
	AllowMethods     []string `mapstructure:"allow_methods"`
	AllowHeaders     []string `mapstructure:"allow_headers"`
	AllowCredentials bool     `mapstructure:"allow_credentials"`
}

// RateLimitConfig contains rate limiting configuration
type RateLimitConfig struct {
	Enabled bool `mapstructure:"enabled"`
	RPS     int  `mapstructure:"rps"`     // requests per second
	Burst   int  `mapstructure:"burst"`   // burst size
}

// JWTConfig contains JWT configuration
type JWTConfig struct {
	Secret     string        `mapstructure:"secret"`
	Expiration time.Duration `mapstructure:"expiration"`
	Issuer     string        `mapstructure:"issuer"`
}

// Load loads configuration from file and environment variables
func Load(configFile string) (*Config, error) {
	// Set defaults
	setDefaults()

	// Set config file
	if configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		viper.SetConfigName("gateway")
		viper.SetConfigType("yaml")
		viper.AddConfigPath("./configs")
		viper.AddConfigPath("../configs")
		viper.AddConfigPath("/etc/isa_cloud")
	}

	// Read environment variables
	viper.AutomaticEnv()
	viper.SetEnvPrefix("ISA_CLOUD")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found, use defaults and env vars
	}

	// Unmarshal config
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

// setDefaults sets default configuration values
func setDefaults() {
	// App
	viper.SetDefault("app.name", "IsA Cloud Gateway")
	viper.SetDefault("app.version", "1.0.0")

	// Environment
	viper.SetDefault("environment", "development")
	viper.SetDefault("debug", false)

	// Server
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.http_port", 8000)
	viper.SetDefault("server.grpc_port", 9000)

	// Services
	viper.SetDefault("services.user_service.host", "localhost")
	viper.SetDefault("services.user_service.http_port", 8100)
	viper.SetDefault("services.user_service.grpc_port", 9100)
	viper.SetDefault("services.user_service.timeout", "30s")
	viper.SetDefault("services.user_service.retry.max_attempts", 3)
	viper.SetDefault("services.user_service.retry.backoff", "1s")

	viper.SetDefault("services.auth_service.host", "localhost")
	viper.SetDefault("services.auth_service.http_port", 8101)
	viper.SetDefault("services.auth_service.grpc_port", 9101)
	viper.SetDefault("services.auth_service.timeout", "10s")

	viper.SetDefault("services.agent_service.host", "localhost")
	viper.SetDefault("services.agent_service.http_port", 8080)
	viper.SetDefault("services.agent_service.grpc_port", 9080)
	viper.SetDefault("services.agent_service.timeout", "60s")

	viper.SetDefault("services.model_service.host", "localhost")
	viper.SetDefault("services.model_service.http_port", 8082)
	viper.SetDefault("services.model_service.grpc_port", 9082)
	viper.SetDefault("services.model_service.timeout", "120s")

	viper.SetDefault("services.mcp_service.host", "localhost")
	viper.SetDefault("services.mcp_service.http_port", 8081)
	viper.SetDefault("services.mcp_service.grpc_port", 9081)
	viper.SetDefault("services.mcp_service.timeout", "30s")

	// Database
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.database", "isa_cloud")
	viper.SetDefault("database.username", "postgres")
	viper.SetDefault("database.password", "")
	viper.SetDefault("database.ssl_mode", "disable")

	// Redis
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.database", 0)

	// Logging
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "json")
	viper.SetDefault("logging.output", "stdout")

	// Monitoring
	viper.SetDefault("monitoring.enabled", true)
	viper.SetDefault("monitoring.port", 9090)
	viper.SetDefault("monitoring.path", "/metrics")
	viper.SetDefault("monitoring.namespace", "isa_cloud")
	viper.SetDefault("monitoring.subsystem", "gateway")

	// Security
	viper.SetDefault("security.cors.enabled", true)
	viper.SetDefault("security.cors.allow_origins", []string{"*"})
	viper.SetDefault("security.cors.allow_methods", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})
	viper.SetDefault("security.cors.allow_headers", []string{"*"})
	viper.SetDefault("security.cors.allow_credentials", true)

	viper.SetDefault("security.rate_limit.enabled", true)
	viper.SetDefault("security.rate_limit.rps", 100)
	viper.SetDefault("security.rate_limit.burst", 200)

	viper.SetDefault("security.jwt.secret", "your-secret-key")
	viper.SetDefault("security.jwt.expiration", "24h")
	viper.SetDefault("security.jwt.issuer", "isa-cloud")

	// Blockchain (temporarily disabled)
	// viper.SetDefault("blockchain.enabled", true)
	// viper.SetDefault("blockchain.rpc_endpoint", "http://localhost:8545")
	// viper.SetDefault("blockchain.chain_id", 31337)
	// viper.SetDefault("blockchain.network_name", "localhost")
	// viper.SetDefault("blockchain.private_key", "")
	// viper.SetDefault("blockchain.public_key", "")
	// viper.SetDefault("blockchain.gas_limit", 300000)
	// viper.SetDefault("blockchain.gas_price", "20000000000")
	// viper.SetDefault("blockchain.confirmations", 1)
	// viper.SetDefault("blockchain.health_check_interval", "30s")
}