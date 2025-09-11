package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config представляет конфигурацию Notification Service
type Config struct {
	Server              ServerConfig              `yaml:"server"`
	Health              HealthConfig              `yaml:"health"`
	Metrics             MetricsConfig             `yaml:"metrics"`
	Database            DatabaseConfig            `yaml:"database"`
	Notifications       NotificationsConfig       `yaml:"notifications"`
	WebSocket           WebSocketConfig           `yaml:"websocket"`
	Services            ServicesConfig            `yaml:"services"`
	PushNotifications   PushNotificationsConfig   `yaml:"push_notifications"`
	EmailNotifications  EmailNotificationsConfig  `yaml:"email_notifications"`
	SMSNotifications    SMSNotificationsConfig    `yaml:"sms_notifications"`
	Templates           TemplatesConfig           `yaml:"templates"`
	Events              EventsConfig              `yaml:"events"`
	Logging             LoggingConfig             `yaml:"logging"`
	Environment         string                    `yaml:"-"`
}

// ServerConfig конфигурация HTTP сервера
type ServerConfig struct {
	Port         int           `yaml:"port"`
	Host         string        `yaml:"host"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout"`
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

// DatabaseConfig конфигурация PostgreSQL
type DatabaseConfig struct {
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	Name            string        `yaml:"name"`
	User            string        `yaml:"user"`
	Password        string        `yaml:"password"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `yaml:"conn_max_idle_time"`
	SSLMode         string        `yaml:"ssl_mode"`
}

// NotificationsConfig конфигурация уведомлений
type NotificationsConfig struct {
	MaxNotificationsPerUser int                    `yaml:"max_notifications_per_user"`
	NotificationTTLDays     int                    `yaml:"notification_ttl_days"`
	MaxNotificationText     int                    `yaml:"max_notification_text"`
	AutoCleanupEnabled      bool                   `yaml:"auto_cleanup_enabled"`
	AutoCleanupInterval     time.Duration          `yaml:"auto_cleanup_interval"`
	Types                   map[string]string      `yaml:"types"`
}

// WebSocketConfig конфигурация WebSocket соединения с API Gateway
type WebSocketConfig struct {
	GatewayURL            string        `yaml:"gateway_url"`
	ConnectTimeout        time.Duration `yaml:"connect_timeout"`
	ReadTimeout           time.Duration `yaml:"read_timeout"`
	WriteTimeout          time.Duration `yaml:"write_timeout"`
	ReconnectEnabled      bool          `yaml:"reconnect_enabled"`
	ReconnectInterval     time.Duration `yaml:"reconnect_interval"`
	MaxReconnectAttempts  int           `yaml:"max_reconnect_attempts"`
	PingInterval          time.Duration `yaml:"ping_interval"`
	PongTimeout           time.Duration `yaml:"pong_timeout"`
}

// ServicesConfig конфигурация микросервисов
type ServicesConfig struct {
	NewsManagement ServiceConfig `yaml:"news_management"`
	Pulse          ServiceConfig `yaml:"pulse"`
}

// ServiceConfig конфигурация отдельного микросервиса
type ServiceConfig struct {
	URL     string        `yaml:"url"`
	Timeout time.Duration `yaml:"timeout"`
}

// PushNotificationsConfig конфигурация push уведомлений
type PushNotificationsConfig struct {
	Enabled  bool           `yaml:"enabled"`
	FCM      FCMConfig      `yaml:"fcm"`
	APNS     APNSConfig     `yaml:"apns"`
	WebPush  WebPushConfig  `yaml:"web_push"`
}

// FCMConfig конфигурация Firebase Cloud Messaging
type FCMConfig struct {
	Enabled   bool   `yaml:"enabled"`
	ServerKey string `yaml:"server_key"`
	ProjectID string `yaml:"project_id"`
}

// APNSConfig конфигурация Apple Push Notification Service
type APNSConfig struct {
	Enabled bool   `yaml:"enabled"`
	KeyFile string `yaml:"key_file"`
	KeyID   string `yaml:"key_id"`
	TeamID  string `yaml:"team_id"`
}

// WebPushConfig конфигурация Web Push
type WebPushConfig struct {
	Enabled          bool   `yaml:"enabled"`
	VAPIDPublicKey   string `yaml:"vapid_public_key"`
	VAPIDPrivateKey  string `yaml:"vapid_private_key"`
}

// EmailNotificationsConfig конфигурация email уведомлений
type EmailNotificationsConfig struct {
	Enabled   bool              `yaml:"enabled"`
	SMTP      SMTPConfig        `yaml:"smtp"`
	Templates map[string]string `yaml:"templates"`
}

// SMTPConfig конфигурация SMTP
type SMTPConfig struct {
	Host      string `yaml:"host"`
	Port      int    `yaml:"port"`
	Username  string `yaml:"username"`
	Password  string `yaml:"password"`
	FromEmail string `yaml:"from_email"`
	FromName  string `yaml:"from_name"`
}

// SMSNotificationsConfig конфигурация SMS уведомлений
type SMSNotificationsConfig struct {
	Enabled  bool          `yaml:"enabled"`
	Provider string        `yaml:"provider"`
	Twilio   TwilioConfig  `yaml:"twilio"`
	SMSRU    SMSRUConfig   `yaml:"sms_ru"`
}

// TwilioConfig конфигурация Twilio
type TwilioConfig struct {
	AccountSID string `yaml:"account_sid"`
	AuthToken  string `yaml:"auth_token"`
	FromNumber string `yaml:"from_number"`
}

// SMSRUConfig конфигурация SMS.RU
type SMSRUConfig struct {
	APIID string `yaml:"api_id"`
}

// TemplatesConfig конфигурация шаблонов уведомлений
type TemplatesConfig struct {
	NewsAlert     TemplateConfig `yaml:"news_alert"`
	PulseUpdate   TemplateConfig `yaml:"pulse_update"`
	SystemMessage TemplateConfig `yaml:"system_message"`
}

// TemplateConfig конфигурация шаблона
type TemplateConfig struct {
	Title string `yaml:"title"`
	Body  string `yaml:"body"`
}

// EventsConfig конфигурация обработки событий
type EventsConfig struct {
	BufferSize         int           `yaml:"buffer_size"`
	WorkerCount        int           `yaml:"worker_count"`
	ProcessingTimeout  time.Duration `yaml:"processing_timeout"`
	Retry              RetryConfig   `yaml:"retry"`
}

// RetryConfig конфигурация повторных попыток
type RetryConfig struct {
	Enabled      bool          `yaml:"enabled"`
	MaxAttempts  int           `yaml:"max_attempts"`
	InitialDelay time.Duration `yaml:"initial_delay"`
	MaxDelay     time.Duration `yaml:"max_delay"`
	Multiplier   float64       `yaml:"multiplier"`
}

// LoggingConfig конфигурация логирования
type LoggingConfig struct {
	Level                    string `yaml:"level"`
	Format                   string `yaml:"format"`
	Output                   string `yaml:"output"`
	FilePath                 string `yaml:"file_path"`
	LogNotifications         bool   `yaml:"log_notifications"`
	SlowOperationThreshold   int    `yaml:"slow_operation_threshold"`
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

	// Database
	if host := os.Getenv("POSTGRES_HOST"); host != "" {
		c.Database.Host = host
	}
	if port := os.Getenv("POSTGRES_PORT"); port != "" {
		if p := parseInt(port, c.Database.Port); p > 0 {
			c.Database.Port = p
		}
	}
	if name := os.Getenv("POSTGRES_DB"); name != "" {
		c.Database.Name = name
	}
	if user := os.Getenv("POSTGRES_USER"); user != "" {
		c.Database.User = user
	}
	if password := os.Getenv("POSTGRES_PASSWORD"); password != "" {
		c.Database.Password = password
	}

	// WebSocket
	if gatewayURL := os.Getenv("WEBSOCKET_GATEWAY_URL"); gatewayURL != "" {
		c.WebSocket.GatewayURL = gatewayURL
	}

	// Services
	if newsURL := os.Getenv("NEWS_MANAGEMENT_SERVICE_URL"); newsURL != "" {
		c.Services.NewsManagement.URL = newsURL
	}
	if pulseURL := os.Getenv("PULSE_SERVICE_URL"); pulseURL != "" {
		c.Services.Pulse.URL = pulseURL
	}

	// Logging
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		c.Logging.Level = level
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

	if c.Database.Name == "" {
		return fmt.Errorf("database name is required")
	}

	if c.Database.User == "" {
		return fmt.Errorf("database user is required")
	}

	if c.WebSocket.GatewayURL == "" {
		return fmt.Errorf("WebSocket gateway URL is required")
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

// GetDatabaseDSN возвращает строку подключения к базе данных
func (c *Config) GetDatabaseDSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.Name,
		c.Database.SSLMode,
	)
}

// GetNotificationTTL возвращает время жизни уведомлений
func (c *Config) GetNotificationTTL() time.Duration {
	return time.Duration(c.Notifications.NotificationTTLDays) * 24 * time.Hour
}

// IsNotificationTypeValid проверяет валидность типа уведомления
func (c *Config) IsNotificationTypeValid(notificationType string) bool {
	_, exists := c.Notifications.Types[notificationType]
	return exists
}

// GetNotificationTypeDescription возвращает описание типа уведомления
func (c *Config) GetNotificationTypeDescription(notificationType string) string {
	if desc, exists := c.Notifications.Types[notificationType]; exists {
		return desc
	}
	return "Неизвестный тип уведомления"
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
