package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"

	"notification-service/internal/config"
)

// Database представляет подключение к базе данных
type Database struct {
	db     *sql.DB
	config *config.Config
	logger *logrus.Logger
}

// New создает новое подключение к базе данных
func New(config *config.Config, logger *logrus.Logger) (*Database, error) {
	// Открываем соединение с базой данных
	db, err := sql.Open("postgres", config.GetDatabaseDSN())
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Настраиваем пул соединений
	db.SetMaxOpenConns(config.Database.MaxOpenConns)
	db.SetMaxIdleConns(config.Database.MaxIdleConns)
	db.SetConnMaxLifetime(config.Database.ConnMaxLifetime)
	db.SetConnMaxIdleTime(config.Database.ConnMaxIdleTime)

	// Проверяем соединение
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	database := &Database{
		db:     db,
		config: config,
		logger: logger,
	}

	// Создаем таблицы если они не существуют
	if err := database.createTables(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	logger.Info("Successfully connected to PostgreSQL database")
	return database, nil
}

// Close закрывает соединение с базой данных
func (d *Database) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

// GetDB возвращает экземпляр *sql.DB
func (d *Database) GetDB() *sql.DB {
	return d.db
}

// Ping проверяет соединение с базой данных
func (d *Database) Ping() error {
	return d.db.Ping()
}

// createTables создает необходимые таблицы
func (d *Database) createTables() error {
	queries := []string{
		// Таблица уведомлений
		`CREATE TABLE IF NOT EXISTS notifications (
			id SERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			type VARCHAR(50) NOT NULL,
			title VARCHAR(200) NOT NULL,
			message TEXT NOT NULL,
			data JSONB,
			is_read BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			read_at TIMESTAMP WITH TIME ZONE,
			expires_at TIMESTAMP WITH TIME ZONE,
			
			-- Индексы
			INDEX idx_notifications_user_id (user_id),
			INDEX idx_notifications_type (type),
			INDEX idx_notifications_is_read (is_read),
			INDEX idx_notifications_created_at (created_at),
			INDEX idx_notifications_expires_at (expires_at)
		)`,
		
		// Таблица настроек уведомлений пользователей
		`CREATE TABLE IF NOT EXISTS user_notification_settings (
			user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
			news_alerts_enabled BOOLEAN DEFAULT TRUE,
			pulse_updates_enabled BOOLEAN DEFAULT TRUE,
			system_messages_enabled BOOLEAN DEFAULT TRUE,
			email_notifications BOOLEAN DEFAULT FALSE,
			push_notifications BOOLEAN DEFAULT FALSE,
			sms_notifications BOOLEAN DEFAULT FALSE,
			quiet_hours_start TIME,
			quiet_hours_end TIME,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)`,
		
		// Таблица устройств для push уведомлений
		`CREATE TABLE IF NOT EXISTS user_devices (
			id SERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			device_type VARCHAR(20) NOT NULL, -- ios, android, web
			device_token VARCHAR(500) NOT NULL,
			is_active BOOLEAN DEFAULT TRUE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			
			-- Уникальный индекс для предотвращения дубликатов
			UNIQUE(user_id, device_token),
			INDEX idx_user_devices_user_id (user_id),
			INDEX idx_user_devices_is_active (is_active)
		)`,
		
		// Таблица шаблонов уведомлений
		`CREATE TABLE IF NOT EXISTS notification_templates (
			id SERIAL PRIMARY KEY,
			type VARCHAR(50) NOT NULL UNIQUE,
			title_template TEXT NOT NULL,
			body_template TEXT NOT NULL,
			variables JSONB,
			priority VARCHAR(20) DEFAULT 'medium',
			expires_in_hours INTEGER,
			is_active BOOLEAN DEFAULT TRUE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			
		)`,
		
		// Индексы для notification_templates
		`CREATE INDEX IF NOT EXISTS idx_notification_templates_type ON notification_templates (type)`,
		`CREATE INDEX IF NOT EXISTS idx_notification_templates_is_active ON notification_templates (is_active)`,
		
		// Таблица логов доставки уведомлений
		`CREATE TABLE IF NOT EXISTS notification_delivery_logs (
			id SERIAL PRIMARY KEY,
			notification_id INTEGER REFERENCES notifications(id) ON DELETE CASCADE,
			delivery_method VARCHAR(20) NOT NULL, -- websocket, email, push, sms
			status VARCHAR(20) NOT NULL, -- pending, sent, delivered, failed
			error_message TEXT,
			attempts INTEGER DEFAULT 0,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)`,
		
		// Индексы для notification_delivery_logs
		`CREATE INDEX IF NOT EXISTS idx_delivery_logs_notification_id ON notification_delivery_logs (notification_id)`,
		`CREATE INDEX IF NOT EXISTS idx_delivery_logs_status ON notification_delivery_logs (status)`,
		`CREATE INDEX IF NOT EXISTS idx_delivery_logs_created_at ON notification_delivery_logs (created_at)`,
	}

	for _, query := range queries {
		if _, err := d.db.Exec(query); err != nil {
			d.logger.WithError(err).WithField("query", query).Error("Failed to create table")
			return fmt.Errorf("failed to execute query: %w", err)
		}
	}

	// Вставляем базовые шаблоны уведомлений
	if err := d.insertDefaultTemplates(); err != nil {
		d.logger.WithError(err).Warn("Failed to insert default notification templates")
	}

	d.logger.Info("Database tables created successfully")
	return nil
}

// insertDefaultTemplates вставляет базовые шаблоны уведомлений
func (d *Database) insertDefaultTemplates() error {
	templates := []struct {
		Type         string
		Title        string
		Body         string
		Variables    string
		Priority     string
		ExpiresHours *int
	}{
		{
			Type:      "news_alert",
			Title:     "Новая важная новость",
			Body:      "{{.Title}}\n\n{{.Summary}}",
			Variables: `{"Title": "Заголовок новости", "Summary": "Краткое содержание"}`,
			Priority:  "medium",
		},
		{
			Type:      "pulse_update",
			Title:     "Обновление пульса \"{{.PulseName}}\"",
			Body:      "Найдено {{.NewsCount}} новых новостей по вашим критериям",
			Variables: `{"PulseName": "Название пульса", "NewsCount": "Количество новостей"}`,
			Priority:  "low",
		},
		{
			Type:      "system_message",
			Title:     "Системное уведомление",
			Body:      "{{.Message}}",
			Variables: `{"Message": "Текст сообщения"}`,
			Priority:  "high",
		},
		{
			Type:         "user_mention",
			Title:        "Вас упомянули",
			Body:         "{{.Author}} упомянул вас в {{.Context}}",
			Variables:    `{"Author": "Автор", "Context": "Контекст"}`,
			Priority:     "medium",
			ExpiresHours: intPtr(72), // 3 дня
		},
	}

	for _, template := range templates {
		query := `
			INSERT INTO notification_templates 
			(type, title_template, body_template, variables, priority, expires_in_hours) 
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (type) DO NOTHING`
		
		_, err := d.db.Exec(query,
			template.Type,
			template.Title,
			template.Body,
			template.Variables,
			template.Priority,
			template.ExpiresHours,
		)
		if err != nil {
			return fmt.Errorf("failed to insert template %s: %w", template.Type, err)
		}
	}

	return nil
}

// GetStats возвращает статистику базы данных
func (d *Database) GetStats() map[string]interface{} {
	stats := d.db.Stats()
	
	return map[string]interface{}{
		"max_open_connections":     stats.MaxOpenConnections,
		"open_connections":         stats.OpenConnections,
		"in_use":                  stats.InUse,
		"idle":                    stats.Idle,
		"wait_count":              stats.WaitCount,
		"wait_duration":           stats.WaitDuration,
		"max_idle_closed":         stats.MaxIdleClosed,
		"max_idle_time_closed":    stats.MaxIdleTimeClosed,
		"max_lifetime_closed":     stats.MaxLifetimeClosed,
	}
}

// CleanupExpiredNotifications удаляет истекшие уведомления
func (d *Database) CleanupExpiredNotifications() error {
	query := `DELETE FROM notifications WHERE expires_at IS NOT NULL AND expires_at < CURRENT_TIMESTAMP`
	
	result, err := d.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to cleanup expired notifications: %w", err)
	}
	
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		d.logger.WithField("deleted_count", rowsAffected).Info("Cleaned up expired notifications")
	}
	
	return nil
}

// CleanupOldNotifications удаляет старые уведомления
func (d *Database) CleanupOldNotifications(olderThan time.Duration) error {
	query := `DELETE FROM notifications WHERE created_at < $1`
	cutoffTime := time.Now().Add(-olderThan)
	
	result, err := d.db.Exec(query, cutoffTime)
	if err != nil {
		return fmt.Errorf("failed to cleanup old notifications: %w", err)
	}
	
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		d.logger.WithField("deleted_count", rowsAffected).Info("Cleaned up old notifications")
	}
	
	return nil
}

// StartCleanupRoutine запускает регулярную очистку уведомлений
func (d *Database) StartCleanupRoutine() {
	if !d.config.Notifications.AutoCleanupEnabled {
		return
	}
	
	ticker := time.NewTicker(d.config.Notifications.AutoCleanupInterval)
	
	go func() {
		defer ticker.Stop()
		
		for range ticker.C {
			// Удаляем истекшие уведомления
			if err := d.CleanupExpiredNotifications(); err != nil {
				d.logger.WithError(err).Error("Failed to cleanup expired notifications")
			}
			
			// Удаляем старые уведомления
			if err := d.CleanupOldNotifications(d.config.GetNotificationTTL()); err != nil {
				d.logger.WithError(err).Error("Failed to cleanup old notifications")
			}
		}
	}()
	
	d.logger.WithField("interval", d.config.Notifications.AutoCleanupInterval).
		Info("Started notification cleanup routine")
}

// Вспомогательные функции
func intPtr(i int) *int {
	return &i
}
