package database

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/sirupsen/logrus"

	"pulse-service/internal/config"
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
	return regexp.MustCompile(`password=\S+`).ReplaceAllString(dsn, "password=***")
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
	
	// Проверка работы с данными пульсов
	var pulseCount int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM user_pulses WHERE is_active = true").Scan(&pulseCount); err != nil {
		result["query_error"] = err.Error()
	} else {
		result["active_pulses_count"] = pulseCount
	}
	
	// Проверка работы с данными связей пульсов и источников
	var pulseSourceCount int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM pulse_sources").Scan(&pulseSourceCount); err != nil {
		result["pulse_sources_query_error"] = err.Error()
	} else {
		result["pulse_sources_count"] = pulseSourceCount
	}
	
	return result
}

// Вспомогательные функции для работы с контекстом

// WithTimeout создает контекст с таймаутом для запросов к БД
func WithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

// Migrate проверяет наличие необходимых таблиц для работы с пульсами
func (db *DB) Migrate() error {
	db.logger.Info("Checking database schema for pulse service")
	
	// Проверяем, что основные таблицы существуют
	tables := []string{
		"users",           // Пользователи (создается отдельно)
		"categories",      // Категории новостей
		"news_sources",    // Источники новостей
		"news",           // Новости
		"user_pulses",    // Пульсы пользователей
		"pulse_sources",  // Связь пульсов с источниками
		"pulse_categories", // Связь пульсов с категориями
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
	
	db.logger.Info("Database schema check completed for pulse service")
	return nil
}

// CreatePulseIndexes создает дополнительные индексы для оптимизации работы с пульсами
func (db *DB) CreatePulseIndexes() error {
	db.logger.Info("Creating pulse-specific database indexes")
	
	indexes := []string{
		// Индекс для быстрого поиска пульсов пользователя
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_user_pulses_user_id_active 
		 ON user_pulses(user_id, is_active) WHERE is_active = true`,
		
		// Индекс для поиска дефолтных пульсов
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_user_pulses_default 
		 ON user_pulses(is_default, user_id) WHERE is_default = true`,
		
		// Индекс для сортировки пульсов по времени обновления
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_user_pulses_updated_at 
		 ON user_pulses(updated_at DESC)`,
		
		// Индекс для быстрого поиска источников пульса
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_pulse_sources_pulse_id 
		 ON pulse_sources(pulse_id)`,
		
		// Индекс для быстрого поиска категорий пульса
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_pulse_categories_pulse_id 
		 ON pulse_categories(pulse_id)`,
		
		// Составной индекс для поиска новостей по источникам и времени
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_news_source_published 
		 ON news(source_id, published_at DESC) WHERE is_active = true`,
		
		// Составной индекс для поиска новостей по категориям и времени
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_news_category_published 
		 ON news(category_id, published_at DESC) WHERE is_active = true AND category_id IS NOT NULL`,
		
		// Индекс для полнотекстового поиска по названиям пульсов
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_user_pulses_search 
		 ON user_pulses USING gin(to_tsvector('russian', name || ' ' || COALESCE(description, '')))`,
	}
	
	for _, indexSQL := range indexes {
		if _, err := db.Exec(indexSQL); err != nil {
			db.logger.WithError(err).WithField("sql", indexSQL).Warn("Failed to create index")
			// Не возвращаем ошибку, так как индексы могут уже существовать
		}
	}
	
	db.logger.Info("Pulse-specific indexes creation completed")
	return nil
}
