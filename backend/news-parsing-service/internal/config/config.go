package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config представляет конфигурацию сервиса парсинга новостей
type Config struct {
	Server         ServerConfig   `yaml:"server"`
	Database       DatabaseConfig `yaml:"database"`
	Parsing        ParsingConfig  `yaml:"parsing"`
	Logging        LoggingConfig  `yaml:"logging"`
	Health         HealthConfig   `yaml:"health"`
	Metrics        MetricsConfig  `yaml:"metrics"`
	OpenAIAPIKey   string         `yaml:"openai_api_key"`
	DeepSeekAPIKey string         `yaml:"deepseek_api_key"`
	Environment    string         `yaml:"-"`
}

// ServerConfig конфигурация HTTP сервера
type ServerConfig struct {
	Port         int           `yaml:"port"`
	Host         string        `yaml:"host"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout"`
}

// DatabaseConfig конфигурация базы данных
type DatabaseConfig struct {
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	User            string        `yaml:"user"`
	Password        string        `yaml:"password"`
	DBName          string        `yaml:"dbname"`
	SSLMode         string        `yaml:"sslmode"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
}

// ParsingConfig конфигурация парсинга RSS
type ParsingConfig struct {
	Interval             time.Duration `yaml:"interval"`
	MaxConcurrentParsers int           `yaml:"max_concurrent_parsers"`
	RequestTimeout       time.Duration `yaml:"request_timeout"`
	UserAgent            string        `yaml:"user_agent"`
	MaxFeedSize          int64         `yaml:"max_feed_size"`
	BatchSize            int           `yaml:"batch_size"`
	EnableDeduplication  bool          `yaml:"enable_deduplication"`
	MinTitleLength       int           `yaml:"min_title_length"`
	MaxTitleLength       int           `yaml:"max_title_length"`
}

// LoggingConfig конфигурация логирования
type LoggingConfig struct {
	Level    string `yaml:"level"`
	Format   string `yaml:"format"`
	Output   string `yaml:"output"`
	FilePath string `yaml:"file_path"`
}

// HealthConfig конфигурация health check
type HealthConfig struct {
	Port int    `yaml:"port"`
	Path string `yaml:"path"`
}

// MetricsConfig конфигурация метрик
type MetricsConfig struct {
	Enabled bool   `yaml:"enabled"`
	Port    int    `yaml:"port"`
	Path    string `yaml:"path"`
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
	// Database
	if host := os.Getenv("POSTGRES_HOST"); host != "" {
		c.Database.Host = host
	}
	if port := os.Getenv("POSTGRES_PORT"); port != "" {
		if p := parseInt(port, c.Database.Port); p > 0 {
			c.Database.Port = p
		}
	}
	if user := os.Getenv("POSTGRES_USER"); user != "" {
		c.Database.User = user
	}
	if password := os.Getenv("POSTGRES_PASSWORD"); password != "" {
		c.Database.Password = password
	}
	if dbname := os.Getenv("POSTGRES_DB"); dbname != "" {
		c.Database.DBName = dbname
	}
	if sslmode := os.Getenv("POSTGRES_SSL_MODE"); sslmode != "" {
		c.Database.SSLMode = sslmode
	}

	// Server
	if port := os.Getenv("APP_PORT"); port != "" {
		if p := parseInt(port, c.Server.Port); p > 0 {
			c.Server.Port = p
		}
	}
	if host := os.Getenv("APP_HOST"); host != "" {
		c.Server.Host = host
	}

	// Parsing
	if interval := os.Getenv("RSS_PARSE_INTERVAL_MINUTES"); interval != "" {
		if d, err := time.ParseDuration(interval + "m"); err == nil {
			c.Parsing.Interval = d
		}
	}
	if timeout := os.Getenv("RSS_REQUEST_TIMEOUT_SECONDS"); timeout != "" {
		if d, err := time.ParseDuration(timeout + "s"); err == nil {
			c.Parsing.RequestTimeout = d
		}
	}
	if userAgent := os.Getenv("RSS_USER_AGENT"); userAgent != "" {
		c.Parsing.UserAgent = userAgent
	}
	if maxParsers := os.Getenv("RSS_MAX_CONCURRENT_PARSERS"); maxParsers != "" {
		if p := parseInt(maxParsers, c.Parsing.MaxConcurrentParsers); p > 0 {
			c.Parsing.MaxConcurrentParsers = p
		}
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

	// OpenAI API Key
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		c.OpenAIAPIKey = apiKey
	}

	// DeepSeek API Key
	if apiKey := os.Getenv("DEEPSEEK_API_KEY"); apiKey != "" {
		c.DeepSeekAPIKey = apiKey
	}
}

// validate проверяет корректность конфигурации
func (c *Config) validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}

	if c.Database.User == "" {
		return fmt.Errorf("database user is required")
	}

	if c.Database.DBName == "" {
		return fmt.Errorf("database name is required")
	}

	if c.Parsing.Interval <= 0 {
		return fmt.Errorf("parsing interval must be positive")
	}

	if c.Parsing.MaxConcurrentParsers <= 0 {
		return fmt.Errorf("max concurrent parsers must be positive")
	}

	if c.Parsing.RequestTimeout <= 0 {
		return fmt.Errorf("request timeout must be positive")
	}

	return nil
}

// GetDSN возвращает строку подключения к базе данных
func (c *Config) GetDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.DBName,
		c.Database.SSLMode,
	)
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
