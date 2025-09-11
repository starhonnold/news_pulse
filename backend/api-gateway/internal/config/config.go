package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config представляет конфигурацию API Gateway
type Config struct {
	Server      ServerConfig      `yaml:"server"`
	Services    ServicesConfig    `yaml:"services"`
	Auth        AuthConfig        `yaml:"auth"`
	RateLimiting RateLimitingConfig `yaml:"rate_limiting"`
	CORS        CORSConfig        `yaml:"cors"`
	WebSocket   WebSocketConfig   `yaml:"websocket"`
	Logging     LoggingConfig     `yaml:"logging"`
	Health      HealthConfig      `yaml:"health"`
	Metrics     MetricsConfig     `yaml:"metrics"`
	Proxy       ProxyConfig       `yaml:"proxy"`
	Environment string            `yaml:"-"`
}

// ServerConfig конфигурация HTTP сервера
type ServerConfig struct {
	Port         int           `yaml:"port"`
	Host         string        `yaml:"host"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout"`
}

// ServicesConfig конфигурация микросервисов
type ServicesConfig struct {
	NewsParsing    ServiceConfig `yaml:"news_parsing"`
	NewsManagement ServiceConfig `yaml:"news_management"`
	Pulse          ServiceConfig `yaml:"pulse"`
}

// ServiceConfig конфигурация отдельного микросервиса
type ServiceConfig struct {
	Name           string        `yaml:"name"`
	URL            string        `yaml:"url"`
	HealthEndpoint string        `yaml:"health_endpoint"`
	Timeout        time.Duration `yaml:"timeout"`
	RetryAttempts  int           `yaml:"retry_attempts"`
}

// AuthConfig конфигурация аутентификации
type AuthConfig struct {
	JWTSecret                string   `yaml:"jwt_secret"`
	JWTExpirationHours       int      `yaml:"jwt_expiration_hours"`
	JWTRefreshExpirationHours int     `yaml:"jwt_refresh_expiration_hours"`
	Enabled                  bool     `yaml:"enabled"`
	PublicRoutes             []string `yaml:"public_routes"`
}

// RateLimitingConfig конфигурация rate limiting
type RateLimitingConfig struct {
	Enabled      bool           `yaml:"enabled"`
	Global       RateLimitRule  `yaml:"global"`
	PerUser      RateLimitRule  `yaml:"per_user"`
	Anonymous    RateLimitRule  `yaml:"anonymous"`
	WhitelistIPs []string       `yaml:"whitelist_ips"`
}

// RateLimitRule правило rate limiting
type RateLimitRule struct {
	RequestsPerMinute int `yaml:"requests_per_minute"`
	Burst            int `yaml:"burst"`
}

// CORSConfig конфигурация CORS
type CORSConfig struct {
	Enabled          bool     `yaml:"enabled"`
	AllowedOrigins   []string `yaml:"allowed_origins"`
	AllowedMethods   []string `yaml:"allowed_methods"`
	AllowedHeaders   []string `yaml:"allowed_headers"`
	ExposedHeaders   []string `yaml:"expose_headers"`
	AllowCredentials bool     `yaml:"allow_credentials"`
	MaxAge           int      `yaml:"max_age"`
}

// WebSocketConfig конфигурация WebSocket
type WebSocketConfig struct {
	Enabled                bool          `yaml:"enabled"`
	Path                   string        `yaml:"path"`
	ReadBufferSize         int           `yaml:"read_buffer_size"`
	WriteBufferSize        int           `yaml:"write_buffer_size"`
	HandshakeTimeout       time.Duration `yaml:"handshake_timeout"`
	MaxConnections         int           `yaml:"max_connections"`
	MaxConnectionsPerUser  int           `yaml:"max_connections_per_user"`
	PingPeriod            time.Duration `yaml:"ping_period"`
	PongWait              time.Duration `yaml:"pong_wait"`
	WriteWait             time.Duration `yaml:"write_wait"`
}

// LoggingConfig конфигурация логирования
type LoggingConfig struct {
	Level                string `yaml:"level"`
	Format               string `yaml:"format"`
	Output               string `yaml:"output"`
	FilePath             string `yaml:"file_path"`
	AccessLog            bool   `yaml:"access_log"`
	SlowRequestThreshold int    `yaml:"slow_request_threshold"`
}

// HealthConfig конфигурация health check
type HealthConfig struct {
	Port                int           `yaml:"port"`
	Path                string        `yaml:"path"`
	CheckServices       bool          `yaml:"check_services"`
	ServiceCheckTimeout time.Duration `yaml:"service_check_timeout"`
}

// MetricsConfig конфигурация метрик
type MetricsConfig struct {
	Enabled bool   `yaml:"enabled"`
	Port    int    `yaml:"port"`
	Path    string `yaml:"path"`
}

// ProxyConfig конфигурация прокси
type ProxyConfig struct {
	DialTimeout             time.Duration         `yaml:"dial_timeout"`
	ResponseHeaderTimeout   time.Duration         `yaml:"response_header_timeout"`
	DisableCompression      bool                  `yaml:"disable_compression"`
	FlushInterval          time.Duration         `yaml:"flush_interval"`
	AddHeaders             map[string]string     `yaml:"add_headers"`
	RemoveHeaders          []string              `yaml:"remove_headers"`
}

// LoadConfig загружает конфигурацию из файла и переменных окружения
func LoadConfig(configPath string) (*Config, error) {
	// Читаем файл конфигурации
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Переопределяем значения из переменных окружения
	config.overrideFromEnv()

	// Устанавливаем окружение
	config.Environment = getEnv("APP_ENV", "development")

	// Валидируем конфигурацию
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

// overrideFromEnv переопределяет значения конфигурации из переменных окружения
func (c *Config) overrideFromEnv() {
	// Server
	if port := os.Getenv("APP_PORT"); port != "" {
		if p := parseInt(port, c.Server.Port); p > 0 {
			c.Server.Port = p
		}
	}
	if host := os.Getenv("APP_HOST"); host != "" {
		c.Server.Host = host
	}

	// Services URLs
	if url := os.Getenv("NEWS_PARSING_SERVICE_URL"); url != "" {
		c.Services.NewsParsing.URL = url
	}
	if url := os.Getenv("NEWS_MANAGEMENT_SERVICE_URL"); url != "" {
		c.Services.NewsManagement.URL = url
	}
	if url := os.Getenv("PULSE_SERVICE_URL"); url != "" {
		c.Services.Pulse.URL = url
	}

	// Auth
	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		c.Auth.JWTSecret = secret
	}
	if enabled := os.Getenv("AUTH_ENABLED"); enabled != "" {
		c.Auth.Enabled = enabled == "true"
	}

	// Rate Limiting
	if enabled := os.Getenv("RATE_LIMITING_ENABLED"); enabled != "" {
		c.RateLimiting.Enabled = enabled == "true"
	}

	// CORS
	if enabled := os.Getenv("CORS_ENABLED"); enabled != "" {
		c.CORS.Enabled = enabled == "true"
	}

	// WebSocket
	if enabled := os.Getenv("WEBSOCKET_ENABLED"); enabled != "" {
		c.WebSocket.Enabled = enabled == "true"
	}

	// Logging
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		c.Logging.Level = level
	}
	if format := os.Getenv("LOG_FORMAT"); format != "" {
		c.Logging.Format = format
	}
	if output := os.Getenv("LOG_OUTPUT"); output != "" {
		c.Logging.Output = output
	}
}

// validate проверяет корректность конфигурации
func (c *Config) validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	if c.Services.NewsParsing.URL == "" {
		return fmt.Errorf("news parsing service URL is required")
	}

	if c.Services.NewsManagement.URL == "" {
		return fmt.Errorf("news management service URL is required")
	}

	if c.Services.Pulse.URL == "" {
		return fmt.Errorf("pulse service URL is required")
	}

	if c.Auth.Enabled && c.Auth.JWTSecret == "" {
		return fmt.Errorf("JWT secret is required when auth is enabled")
	}

	if c.Auth.JWTExpirationHours <= 0 {
		return fmt.Errorf("JWT expiration hours must be positive")
	}

	return nil
}

// GetServerAddr возвращает адрес сервера
func (c *Config) GetServerAddr() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

// GetHealthAddr возвращает адрес health check сервера
func (c *Config) GetHealthAddr() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Health.Port)
}

// GetMetricsAddr возвращает адрес metrics сервера
func (c *Config) GetMetricsAddr() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Metrics.Port)
}

// IsPublicRoute проверяет, является ли маршрут публичным
func (c *Config) IsPublicRoute(path string) bool {
	for _, route := range c.Auth.PublicRoutes {
		if path == route || matchPattern(path, route) {
			return true
		}
	}
	return false
}

// IsWhitelistedIP проверяет, находится ли IP в белом списке
func (c *Config) IsWhitelistedIP(ip string) bool {
	for _, whitelistIP := range c.RateLimiting.WhitelistIPs {
		if ip == whitelistIP {
			return true
		}
	}
	return false
}

// GetJWTExpiration возвращает время жизни JWT токена
func (c *Config) GetJWTExpiration() time.Duration {
	return time.Duration(c.Auth.JWTExpirationHours) * time.Hour
}

// GetJWTRefreshExpiration возвращает время жизни refresh токена
func (c *Config) GetJWTRefreshExpiration() time.Duration {
	return time.Duration(c.Auth.JWTRefreshExpirationHours) * time.Hour
}

// Вспомогательные функции
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseInt(s string, defaultValue int) int {
	if s == "" {
		return defaultValue
	}
	
	var result int
	if _, err := fmt.Sscanf(s, "%d", &result); err != nil {
		return defaultValue
	}
	return result
}

// matchPattern проверяет, соответствует ли путь паттерну
func matchPattern(path, pattern string) bool {
	// Простая реализация для паттернов вида "/api/*"
	if pattern == "*" {
		return true
	}
	
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(path) >= len(prefix) && path[:len(prefix)] == prefix
	}
	
	return path == pattern
}
