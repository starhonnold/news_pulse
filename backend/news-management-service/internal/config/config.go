package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config представляет конфигурацию сервиса управления новостями
type Config struct {
	Server      ServerConfig   `yaml:"server"`
	Database    DatabaseConfig `yaml:"database"`
	API         APIConfig      `yaml:"api"`
	Caching     CachingConfig  `yaml:"caching"`
	Logging     LoggingConfig  `yaml:"logging"`
	Health      HealthConfig   `yaml:"health"`
	Metrics     MetricsConfig  `yaml:"metrics"`
	Environment string         `yaml:"-"`
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

// APIConfig конфигурация API
type APIConfig struct {
	MaxPageSize          int           `yaml:"max_page_size"`
	DefaultPageSize      int           `yaml:"default_page_size"`
	MaxSearchLength      int           `yaml:"max_search_length"`
	DBTimeout            time.Duration `yaml:"db_timeout"`
	EnableFulltextSearch bool          `yaml:"enable_fulltext_search"`
}

// CachingConfig конфигурация кеширования
type CachingConfig struct {
	Enabled       bool `yaml:"enabled"`
	NewsTTL       int  `yaml:"news_ttl"`
	CategoriesTTL int  `yaml:"categories_ttl"`
	SourcesTTL    int  `yaml:"sources_ttl"`
	MaxSize       int  `yaml:"max_size"`
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

	// API
	if pageSize := os.Getenv("API_DEFAULT_PAGE_SIZE"); pageSize != "" {
		if p := parseInt(pageSize, c.API.DefaultPageSize); p > 0 {
			c.API.DefaultPageSize = p
		}
	}
	if maxPageSize := os.Getenv("API_MAX_PAGE_SIZE"); maxPageSize != "" {
		if p := parseInt(maxPageSize, c.API.MaxPageSize); p > 0 {
			c.API.MaxPageSize = p
		}
	}

	// Caching
	if enabled := os.Getenv("CACHE_ENABLED"); enabled != "" {
		c.Caching.Enabled = enabled == "true"
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

	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}

	if c.Database.User == "" {
		return fmt.Errorf("database user is required")
	}

	if c.Database.DBName == "" {
		return fmt.Errorf("database name is required")
	}

	if c.API.MaxPageSize <= 0 {
		return fmt.Errorf("max page size must be positive")
	}

	if c.API.DefaultPageSize <= 0 || c.API.DefaultPageSize > c.API.MaxPageSize {
		return fmt.Errorf("default page size must be positive and not exceed max page size")
	}

	return nil
}

// GetDSN возвращает строку подключения к базе данных
func (c *Config) GetDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s client_encoding=UTF8",
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
