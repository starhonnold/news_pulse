package database

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/sirupsen/logrus"

	"news-parsing-service/internal/config"
)

// DB представляет обертку над sql.DB с дополнительным функционалом
type DB struct {
	*sql.DB
	logger *logrus.Logger
}

// New создает новое подключение к базе данных
func New(cfg *config.Config, logger *logrus.Logger) (*DB, error) {
	dsn := cfg.GetDSN()

	logger.WithField("dsn", maskPassword(dsn)).Info("Connecting to database")

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Настройка пула соединений
	db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	db.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)

	// Проверка соединения
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("Successfully connected to database")

	return &DB{
		DB:     db,
		logger: logger,
	}, nil
}

// Close закрывает соединение с базой данных
func (db *DB) Close() error {
	db.logger.Info("Closing database connection")
	return db.DB.Close()
}

// Health проверяет состояние базы данных
func (db *DB) Health() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return db.PingContext(ctx)
}

// GetStats возвращает статистику соединений с базой данных
func (db *DB) GetStats() sql.DBStats {
	return db.Stats()
}

// Transaction выполняет функцию в транзакции
func (db *DB) Transaction(fn func(*sql.Tx) error) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				db.logger.WithError(rbErr).Error("Failed to rollback transaction")
			}
		} else {
			if cmErr := tx.Commit(); cmErr != nil {
				db.logger.WithError(cmErr).Error("Failed to commit transaction")
				err = cmErr
			}
		}
	}()

	err = fn(tx)
	return err
}

// maskPassword маскирует пароль в строке подключения для логирования
func maskPassword(dsn string) string {
	// Простая замена пароля на звездочки
	// В реальном проекте можно использовать более сложную логику
	return regexp.MustCompile(`password=\S+`).ReplaceAllString(dsn, "password=***")
}

// Миграции и инициализация

// Migrate выполняет миграции базы данных
func (db *DB) Migrate() error {
	db.logger.Info("Running database migrations")

	// Проверяем, что основные таблицы существуют
	tables := []string{
		"countries",
		"categories",
		"news_sources",
		"news",
		"tags",
		"news_tags",
		"parsing_logs",
	}

	for _, table := range tables {
		var exists bool
		query := `
			SELECT EXISTS (
				SELECT FROM information_schema.tables 
				WHERE table_schema = 'public' 
				AND table_name = $1
			)`

		if err := db.QueryRow(query, table).Scan(&exists); err != nil {
			return fmt.Errorf("failed to check table %s: %w", table, err)
		}

		if !exists {
			return fmt.Errorf("table %s does not exist, please run schema initialization first", table)
		}

		db.logger.WithField("table", table).Debug("Table exists")
	}

	db.logger.Info("Database migration completed")
	return nil
}

// InitializeExtensions инициализирует необходимые расширения PostgreSQL
func (db *DB) InitializeExtensions() error {
	db.logger.Info("Initializing PostgreSQL extensions")

	extensions := []string{
		"CREATE EXTENSION IF NOT EXISTS pg_trgm",
		"CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"",
	}

	for _, ext := range extensions {
		if _, err := db.Exec(ext); err != nil {
			db.logger.WithError(err).WithField("extension", ext).Warn("Failed to create extension")
			// Не возвращаем ошибку, так как расширения могут быть уже созданы
		}
	}

	return nil
}

// HealthCheck выполняет проверку здоровья базы данных
func (db *DB) HealthCheck() map[string]interface{} {
	result := make(map[string]interface{})

	// Проверка соединения
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start := time.Now()
	err := db.PingContext(ctx)
	pingDuration := time.Since(start)

	result["ping_duration_ms"] = pingDuration.Milliseconds()
	result["connected"] = err == nil

	if err != nil {
		result["error"] = err.Error()
		return result
	}

	// Статистика соединений
	stats := db.Stats()
	result["open_connections"] = stats.OpenConnections
	result["in_use"] = stats.InUse
	result["idle"] = stats.Idle
	result["wait_count"] = stats.WaitCount
	result["wait_duration_ms"] = stats.WaitDuration.Milliseconds()
	result["max_idle_closed"] = stats.MaxIdleClosed
	result["max_idle_time_closed"] = stats.MaxIdleTimeClosed
	result["max_lifetime_closed"] = stats.MaxLifetimeClosed

	// Проверка работы с данными
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM news_sources WHERE is_active = true").Scan(&count); err != nil {
		result["query_error"] = err.Error()
	} else {
		result["active_sources"] = count
	}

	return result
}

// Вспомогательные функции для работы с контекстом

// WithTimeout создает контекст с таймаутом для запросов к БД
func WithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

// QueryRowContext выполняет запрос с контекстом и возвращает одну строку
func (db *DB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	db.logger.WithFields(logrus.Fields{
		"query": query,
		"args":  args,
	}).Debug("Executing query")

	return db.DB.QueryRowContext(ctx, query, args...)
}

// QueryContext выполняет запрос с контекстом и возвращает множество строк
func (db *DB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	db.logger.WithFields(logrus.Fields{
		"query": query,
		"args":  args,
	}).Debug("Executing query")

	return db.DB.QueryContext(ctx, query, args...)
}

// ExecContext выполняет запрос с контекстом
func (db *DB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	db.logger.WithFields(logrus.Fields{
		"query": query,
		"args":  args,
	}).Debug("Executing query")

	return db.DB.ExecContext(ctx, query, args...)
}
