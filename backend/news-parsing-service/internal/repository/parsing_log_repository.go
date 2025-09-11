package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"news-parsing-service/internal/database"
	"news-parsing-service/internal/models"
)

// ParsingLogRepository представляет репозиторий для работы с логами парсинга
type ParsingLogRepository struct {
	db     *database.DB
	logger *logrus.Logger
}

// NewParsingLogRepository создает новый репозиторий логов парсинга
func NewParsingLogRepository(db *database.DB, logger *logrus.Logger) *ParsingLogRepository {
	return &ParsingLogRepository{
		db:     db,
		logger: logger,
	}
}

// Create создает новую запись лога парсинга
func (r *ParsingLogRepository) Create(ctx context.Context, log *models.ParsingLog) error {
	query := `
		INSERT INTO parsing_logs (source_id, status, news_count, error_message, execution_time_ms)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`
	
	err := r.db.QueryRowContext(ctx, query,
		log.SourceID, log.Status, log.NewsCount, 
		log.ErrorMessage, log.ExecutionTimeMs,
	).Scan(&log.ID, &log.CreatedAt)
	
	if err != nil {
		return fmt.Errorf("failed to create parsing log: %w", err)
	}
	
	r.logger.WithFields(logrus.Fields{
		"log_id":           log.ID,
		"source_id":        log.SourceID,
		"status":           log.Status,
		"news_count":       log.NewsCount,
		"execution_time_ms": log.ExecutionTimeMs,
	}).Debug("Created parsing log")
	
	return nil
}

// GetBySourceID возвращает логи парсинга по ID источника
func (r *ParsingLogRepository) GetBySourceID(ctx context.Context, sourceID int, limit int) ([]models.ParsingLog, error) {
	query := `
		SELECT pl.id, pl.source_id, pl.status, pl.news_count, pl.error_message, 
			   pl.execution_time_ms, pl.created_at,
			   ns.name as source_name, ns.domain as source_domain
		FROM parsing_logs pl
		JOIN news_sources ns ON pl.source_id = ns.id
		WHERE pl.source_id = $1
		ORDER BY pl.created_at DESC
		LIMIT $2`
	
	rows, err := r.db.QueryContext(ctx, query, sourceID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query parsing logs: %w", err)
	}
	defer rows.Close()
	
	var logs []models.ParsingLog
	for rows.Next() {
		var log models.ParsingLog
		var sourceName, sourceDomain string
		
		err := rows.Scan(
			&log.ID, &log.SourceID, &log.Status, &log.NewsCount,
			&log.ErrorMessage, &log.ExecutionTimeMs, &log.CreatedAt,
			&sourceName, &sourceDomain,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan parsing log: %w", err)
		}
		
		log.Source = &models.NewsSource{
			ID:     log.SourceID,
			Name:   sourceName,
			Domain: sourceDomain,
		}
		
		logs = append(logs, log)
	}
	
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}
	
	return logs, nil
}

// GetRecent возвращает последние логи парсинга
func (r *ParsingLogRepository) GetRecent(ctx context.Context, limit int) ([]models.ParsingLog, error) {
	query := `
		SELECT pl.id, pl.source_id, pl.status, pl.news_count, pl.error_message, 
			   pl.execution_time_ms, pl.created_at,
			   ns.name as source_name, ns.domain as source_domain
		FROM parsing_logs pl
		JOIN news_sources ns ON pl.source_id = ns.id
		ORDER BY pl.created_at DESC
		LIMIT $1`
	
	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent parsing logs: %w", err)
	}
	defer rows.Close()
	
	var logs []models.ParsingLog
	for rows.Next() {
		var log models.ParsingLog
		var sourceName, sourceDomain string
		
		err := rows.Scan(
			&log.ID, &log.SourceID, &log.Status, &log.NewsCount,
			&log.ErrorMessage, &log.ExecutionTimeMs, &log.CreatedAt,
			&sourceName, &sourceDomain,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan parsing log: %w", err)
		}
		
		log.Source = &models.NewsSource{
			ID:     log.SourceID,
			Name:   sourceName,
			Domain: sourceDomain,
		}
		
		logs = append(logs, log)
	}
	
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}
	
	return logs, nil
}

// GetErrors возвращает логи с ошибками за указанный период
func (r *ParsingLogRepository) GetErrors(ctx context.Context, since time.Time, limit int) ([]models.ParsingLog, error) {
	query := `
		SELECT pl.id, pl.source_id, pl.status, pl.news_count, pl.error_message, 
			   pl.execution_time_ms, pl.created_at,
			   ns.name as source_name, ns.domain as source_domain
		FROM parsing_logs pl
		JOIN news_sources ns ON pl.source_id = ns.id
		WHERE pl.status IN ('error', 'timeout') AND pl.created_at >= $1
		ORDER BY pl.created_at DESC
		LIMIT $2`
	
	rows, err := r.db.QueryContext(ctx, query, since, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query error logs: %w", err)
	}
	defer rows.Close()
	
	var logs []models.ParsingLog
	for rows.Next() {
		var log models.ParsingLog
		var sourceName, sourceDomain string
		
		err := rows.Scan(
			&log.ID, &log.SourceID, &log.Status, &log.NewsCount,
			&log.ErrorMessage, &log.ExecutionTimeMs, &log.CreatedAt,
			&sourceName, &sourceDomain,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan parsing log: %w", err)
		}
		
		log.Source = &models.NewsSource{
			ID:     log.SourceID,
			Name:   sourceName,
			Domain: sourceDomain,
		}
		
		logs = append(logs, log)
	}
	
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}
	
	return logs, nil
}

// GetStats возвращает статистику парсинга
func (r *ParsingLogRepository) GetStats(ctx context.Context, since time.Time) (*models.ParsingStats, error) {
	query := `
		SELECT 
			COUNT(*) as total_runs,
			COUNT(*) FILTER (WHERE status = 'success') as successful_runs,
			COUNT(*) FILTER (WHERE status IN ('error', 'timeout')) as failed_runs,
			AVG(execution_time_ms) as avg_parse_time,
			MAX(created_at) as last_parse_time
		FROM parsing_logs
		WHERE created_at >= $1`
	
	var stats models.ParsingStats
	var totalRuns int
	var lastParseTime sql.NullTime
	var avgParseTime sql.NullFloat64
	
	err := r.db.QueryRowContext(ctx, query, since).Scan(
		&totalRuns,
		&stats.SuccessfulRuns,
		&stats.FailedRuns,
		&avgParseTime,
		&lastParseTime,
	)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get parsing stats: %w", err)
	}
	
	if avgParseTime.Valid {
		stats.AvgParseTime = avgParseTime.Float64
	}
	
	if lastParseTime.Valid {
		stats.LastParseTime = &lastParseTime.Time
	}
	
	return &stats, nil
}

// GetSourceStats возвращает статистику парсинга по источникам
func (r *ParsingLogRepository) GetSourceStats(ctx context.Context, since time.Time) (map[int]*models.ParsingStats, error) {
	query := `
		SELECT 
			pl.source_id,
			COUNT(*) as total_runs,
			COUNT(*) FILTER (WHERE pl.status = 'success') as successful_runs,
			COUNT(*) FILTER (WHERE pl.status IN ('error', 'timeout')) as failed_runs,
			SUM(pl.news_count) as total_news,
			AVG(pl.execution_time_ms) as avg_parse_time,
			MAX(pl.created_at) as last_parse_time
		FROM parsing_logs pl
		WHERE pl.created_at >= $1
		GROUP BY pl.source_id`
	
	rows, err := r.db.QueryContext(ctx, query, since)
	if err != nil {
		return nil, fmt.Errorf("failed to query source stats: %w", err)
	}
	defer rows.Close()
	
	stats := make(map[int]*models.ParsingStats)
	
	for rows.Next() {
		var sourceID int
		var stat models.ParsingStats
		var totalRuns int
		var lastParseTime sql.NullTime
		var avgParseTime sql.NullFloat64
		
		err := rows.Scan(
			&sourceID,
			&totalRuns,
			&stat.SuccessfulRuns,
			&stat.FailedRuns,
			&stat.TotalNews,
			&avgParseTime,
			&lastParseTime,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan source stats: %w", err)
		}
		
		if avgParseTime.Valid {
			stat.AvgParseTime = avgParseTime.Float64
		}
		
		if lastParseTime.Valid {
			stat.LastParseTime = &lastParseTime.Time
		}
		
		stats[sourceID] = &stat
	}
	
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}
	
	return stats, nil
}

// CleanupOldLogs удаляет старые логи парсинга
func (r *ParsingLogRepository) CleanupOldLogs(ctx context.Context, retentionDays int) (int, error) {
	query := `
		DELETE FROM parsing_logs
		WHERE created_at < NOW() - INTERVAL '%d days'`
	
	result, err := r.db.ExecContext(ctx, fmt.Sprintf(query, retentionDays))
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup old logs: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	r.logger.WithFields(logrus.Fields{
		"retention_days": retentionDays,
		"deleted_count":  rowsAffected,
	}).Info("Cleaned up old parsing logs")
	
	return int(rowsAffected), nil
}

// LogParsingResult записывает результат парсинга в лог
func (r *ParsingLogRepository) LogParsingResult(ctx context.Context, result models.FeedParseResult) error {
	status := models.ParsingStatusSuccess
	errorMessage := ""
	
	if !result.Success {
		if result.Error != "" {
			status = models.ParsingStatusError
			errorMessage = result.Error
		} else {
			status = models.ParsingStatusTimeout
		}
	}
	
	log := &models.ParsingLog{
		SourceID:        result.SourceID,
		Status:          status,
		NewsCount:       len(result.Items),
		ErrorMessage:    errorMessage,
		ExecutionTimeMs: int(result.ExecutionTime.Milliseconds()),
	}
	
	return r.Create(ctx, log)
}

// GetHealthStatus возвращает статус здоровья парсинга
func (r *ParsingLogRepository) GetHealthStatus(ctx context.Context) (map[string]interface{}, error) {
	// Статистика за последние 24 часа
	since := time.Now().Add(-24 * time.Hour)
	stats, err := r.GetStats(ctx, since)
	if err != nil {
		return nil, fmt.Errorf("failed to get health stats: %w", err)
	}
	
	// Процент успешных парсингов
	var successRate float64
	totalRuns := stats.SuccessfulRuns + stats.FailedRuns
	if totalRuns > 0 {
		successRate = float64(stats.SuccessfulRuns) / float64(totalRuns) * 100
	}
	
	// Определяем статус здоровья
	healthStatus := "healthy"
	if successRate < 50 {
		healthStatus = "critical"
	} else if successRate < 80 {
		healthStatus = "warning"
	}
	
	return map[string]interface{}{
		"status":         healthStatus,
		"success_rate":   fmt.Sprintf("%.1f%%", successRate),
		"successful_runs": stats.SuccessfulRuns,
		"failed_runs":    stats.FailedRuns,
		"avg_parse_time": stats.AvgParseTime,
		"last_parse_time": stats.LastParseTime,
		"period_hours":   24,
	}, nil
}
