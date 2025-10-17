package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config представляет конфигурацию сервиса парсинга новостей
type Config struct {
	Server      ServerConfig   `yaml:"server"`
	Database    DatabaseConfig `yaml:"database"`
	Parsing     ParsingConfig  `yaml:"parsing"`
	Logging     LoggingConfig  `yaml:"logging"`
	Health      HealthConfig   `yaml:"health"`
	Metrics     MetricsConfig  `yaml:"metrics"`
	Proxy       ProxyConfig    `yaml:"proxy"`
	AI          AIConfig       `yaml:"ai"`
	FastText    FastTextConfig `yaml:"fasttext"`
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

// ProxyConfig конфигурация прокси
type ProxyConfig struct {
	Enabled  bool   `yaml:"enabled"`
	URL      string `yaml:"url"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// AIConfig конфигурация AI классификатора (Ollama - ОТКЛЮЧЕН)
type AIConfig struct {
	Enabled             bool          `yaml:"enabled"`
	ModelURL            string        `yaml:"model_url"`
	Timeout             time.Duration `yaml:"timeout"`
	MaxRetries          int           `yaml:"max_retries"`
	RetryDelay          time.Duration `yaml:"retry_delay"`
	BatchSize           int           `yaml:"batch_size"`
	ConfidenceThreshold float64       `yaml:"confidence_threshold"`
	UseFallback         bool          `yaml:"use_fallback"`
}

// FastTextConfig конфигурация FastText классификатора
type FastTextConfig struct {
	Enabled       bool          `yaml:"enabled"`
	ServiceURL    string        `yaml:"service_url"`
	Timeout       time.Duration `yaml:"timeout"`
	MinConfidence float64       `yaml:"min_confidence"`
	UseFallback   bool          `yaml:"use_fallback"`
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

	// Proxy configuration
	if enabled := os.Getenv("PROXY_ENABLED"); enabled == "true" {
		c.Proxy.Enabled = true
	}
	if url := os.Getenv("PROXY_URL"); url != "" {
		c.Proxy.URL = url
	}
	if username := os.Getenv("PROXY_USERNAME"); username != "" {
		c.Proxy.Username = username
	}
	if password := os.Getenv("PROXY_PASSWORD"); password != "" {
		c.Proxy.Password = password
	}

	// AI Configuration (Ollama - DEPRECATED, use FastText instead)
	if enabled := os.Getenv("AI_CLASSIFICATION_ENABLED"); enabled != "" {
		c.AI.Enabled = enabled == "true"
	}
	if modelURL := os.Getenv("AI_MODEL_SERVICE_URL"); modelURL != "" {
		c.AI.ModelURL = modelURL
	}
	if timeout := os.Getenv("AI_TIMEOUT_SECONDS"); timeout != "" {
		if d := parseDuration(timeout+"s", c.AI.Timeout); d > 0 {
			c.AI.Timeout = d
		}
	}
	if maxRetries := os.Getenv("AI_MAX_RETRIES"); maxRetries != "" {
		if mr := parseInt(maxRetries, c.AI.MaxRetries); mr > 0 {
			c.AI.MaxRetries = mr
		}
	}
	if batchSize := os.Getenv("AI_BATCH_SIZE"); batchSize != "" {
		if bs := parseInt(batchSize, c.AI.BatchSize); bs > 0 {
			c.AI.BatchSize = bs
		}
	}
	if confidenceThreshold := os.Getenv("AI_CONFIDENCE_THRESHOLD"); confidenceThreshold != "" {
		if ct := parseFloat(confidenceThreshold, c.AI.ConfidenceThreshold); ct >= 0 {
			c.AI.ConfidenceThreshold = ct
		}
	}

	// FastText Configuration
	if enabled := os.Getenv("FASTTEXT_ENABLED"); enabled != "" {
		c.FastText.Enabled = enabled == "true"
	}
	if serviceURL := os.Getenv("FASTTEXT_SERVICE_URL"); serviceURL != "" {
		c.FastText.ServiceURL = serviceURL
	}
	if timeout := os.Getenv("FASTTEXT_TIMEOUT_SECONDS"); timeout != "" {
		if d := parseDuration(timeout+"s", c.FastText.Timeout); d > 0 {
			c.FastText.Timeout = d
		}
	}
	if minConfidence := os.Getenv("FASTTEXT_MIN_CONFIDENCE"); minConfidence != "" {
		if mc := parseFloat(minConfidence, c.FastText.MinConfidence); mc >= 0 {
			c.FastText.MinConfidence = mc
		}
	}
	if useFallback := os.Getenv("FASTTEXT_USE_FALLBACK"); useFallback != "" {
		c.FastText.UseFallback = useFallback == "true"
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

func parseDuration(s string, defaultValue time.Duration) time.Duration {
	if s == "" {
		return defaultValue
	}

	d, err := time.ParseDuration(s)
	if err != nil {
		return defaultValue
	}
	return d
}

func parseFloat(s string, defaultValue float64) float64 {
	if s == "" {
		return defaultValue
	}

	var result float64
	if _, err := fmt.Sscanf(s, "%f", &result); err != nil {
		return defaultValue
	}
	return result
}
