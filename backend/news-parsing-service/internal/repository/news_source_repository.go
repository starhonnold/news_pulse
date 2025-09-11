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

// NewsSourceRepository представляет репозиторий для работы с источниками новостей
type NewsSourceRepository struct {
	db     *database.DB
	logger *logrus.Logger
}

// NewNewsSourceRepository создает новый репозиторий источников новостей
func NewNewsSourceRepository(db *database.DB, logger *logrus.Logger) *NewsSourceRepository {
	return &NewsSourceRepository{
		db:     db,
		logger: logger,
	}
}

// GetAll возвращает все источники новостей
func (r *NewsSourceRepository) GetAll(ctx context.Context) ([]models.NewsSource, error) {
	query := `
		SELECT id, name, domain, rss_url, website_url, country_id, language, 
			   description, logo_url, is_active, last_parsed_at, 
			   parse_interval_minutes, created_at, updated_at
		FROM news_sources
		ORDER BY name`
	
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query news sources: %w", err)
	}
	defer rows.Close()
	
	var sources []models.NewsSource
	for rows.Next() {
		var source models.NewsSource
		err := rows.Scan(
			&source.ID, &source.Name, &source.Domain, &source.RSSURL,
			&source.WebsiteURL, &source.CountryID, &source.Language,
			&source.Description, &source.LogoURL, &source.IsActive,
			&source.LastParsedAt, &source.ParseIntervalMinutes,
			&source.CreatedAt, &source.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan news source: %w", err)
		}
		sources = append(sources, source)
	}
	
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}
	
	return sources, nil
}

// GetActive возвращает все активные источники новостей
func (r *NewsSourceRepository) GetActive(ctx context.Context) ([]models.NewsSource, error) {
	query := `
		SELECT id, name, domain, rss_url, website_url, country_id, language, 
			   description, logo_url, is_active, last_parsed_at, 
			   parse_interval_minutes, created_at, updated_at
		FROM news_sources
		WHERE is_active = true
		ORDER BY name`
	
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query active news sources: %w", err)
	}
	defer rows.Close()
	
	var sources []models.NewsSource
	for rows.Next() {
		var source models.NewsSource
		err := rows.Scan(
			&source.ID, &source.Name, &source.Domain, &source.RSSURL,
			&source.WebsiteURL, &source.CountryID, &source.Language,
			&source.Description, &source.LogoURL, &source.IsActive,
			&source.LastParsedAt, &source.ParseIntervalMinutes,
			&source.CreatedAt, &source.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan news source: %w", err)
		}
		sources = append(sources, source)
	}
	
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}
	
	return sources, nil
}

// GetByID возвращает источник новостей по ID
func (r *NewsSourceRepository) GetByID(ctx context.Context, id int) (*models.NewsSource, error) {
	query := `
		SELECT id, name, domain, rss_url, website_url, country_id, language, 
			   description, logo_url, is_active, last_parsed_at, 
			   parse_interval_minutes, created_at, updated_at
		FROM news_sources
		WHERE id = $1`
	
	var source models.NewsSource
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&source.ID, &source.Name, &source.Domain, &source.RSSURL,
		&source.WebsiteURL, &source.CountryID, &source.Language,
		&source.Description, &source.LogoURL, &source.IsActive,
		&source.LastParsedAt, &source.ParseIntervalMinutes,
		&source.CreatedAt, &source.UpdatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("news source with id %d not found", id)
		}
		return nil, fmt.Errorf("failed to get news source: %w", err)
	}
	
	return &source, nil
}

// GetSourcesToParse возвращает источники, которые нужно парсить
func (r *NewsSourceRepository) GetSourcesToParse(ctx context.Context) ([]models.NewsSource, error) {
	query := `
		SELECT id, name, domain, rss_url, website_url, country_id, language, 
			   description, logo_url, is_active, last_parsed_at, 
			   parse_interval_minutes, created_at, updated_at
		FROM news_sources
		WHERE is_active = true 
		  AND (
		    last_parsed_at IS NULL 
		    OR last_parsed_at < NOW() - INTERVAL '1 minute' * parse_interval_minutes
		  )
		ORDER BY 
		  CASE WHEN last_parsed_at IS NULL THEN 0 ELSE 1 END,
		  last_parsed_at ASC`
	
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query sources to parse: %w", err)
	}
	defer rows.Close()
	
	var sources []models.NewsSource
	for rows.Next() {
		var source models.NewsSource
		err := rows.Scan(
			&source.ID, &source.Name, &source.Domain, &source.RSSURL,
			&source.WebsiteURL, &source.CountryID, &source.Language,
			&source.Description, &source.LogoURL, &source.IsActive,
			&source.LastParsedAt, &source.ParseIntervalMinutes,
			&source.CreatedAt, &source.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan news source: %w", err)
		}
		sources = append(sources, source)
	}
	
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}
	
	r.logger.WithField("count", len(sources)).Debug("Found sources to parse")
	return sources, nil
}

// UpdateLastParsedAt обновляет время последнего парсинга источника
func (r *NewsSourceRepository) UpdateLastParsedAt(ctx context.Context, sourceID int, parsedAt time.Time) error {
	query := `
		UPDATE news_sources 
		SET last_parsed_at = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2`
	
	result, err := r.db.ExecContext(ctx, query, parsedAt, sourceID)
	if err != nil {
		return fmt.Errorf("failed to update last_parsed_at: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("news source with id %d not found", sourceID)
	}
	
	r.logger.WithFields(logrus.Fields{
		"source_id": sourceID,
		"parsed_at": parsedAt,
	}).Debug("Updated last_parsed_at")
	
	return nil
}

// Create создает новый источник новостей
func (r *NewsSourceRepository) Create(ctx context.Context, source *models.NewsSource) error {
	query := `
		INSERT INTO news_sources (name, domain, rss_url, website_url, country_id, 
								 language, description, logo_url, parse_interval_minutes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at`
	
	err := r.db.QueryRowContext(ctx, query,
		source.Name, source.Domain, source.RSSURL, source.WebsiteURL,
		source.CountryID, source.Language, source.Description,
		source.LogoURL, source.ParseIntervalMinutes,
	).Scan(&source.ID, &source.CreatedAt, &source.UpdatedAt)
	
	if err != nil {
		return fmt.Errorf("failed to create news source: %w", err)
	}
	
	r.logger.WithFields(logrus.Fields{
		"source_id":   source.ID,
		"source_name": source.Name,
	}).Info("Created news source")
	
	return nil
}

// Update обновляет источник новостей
func (r *NewsSourceRepository) Update(ctx context.Context, source *models.NewsSource) error {
	query := `
		UPDATE news_sources 
		SET name = $1, domain = $2, rss_url = $3, website_url = $4, 
			country_id = $5, language = $6, description = $7, logo_url = $8,
			parse_interval_minutes = $9, is_active = $10, updated_at = CURRENT_TIMESTAMP
		WHERE id = $11`
	
	result, err := r.db.ExecContext(ctx, query,
		source.Name, source.Domain, source.RSSURL, source.WebsiteURL,
		source.CountryID, source.Language, source.Description, source.LogoURL,
		source.ParseIntervalMinutes, source.IsActive, source.ID,
	)
	
	if err != nil {
		return fmt.Errorf("failed to update news source: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("news source with id %d not found", source.ID)
	}
	
	r.logger.WithFields(logrus.Fields{
		"source_id":   source.ID,
		"source_name": source.Name,
	}).Info("Updated news source")
	
	return nil
}

// Delete удаляет источник новостей (мягкое удаление - устанавливает is_active = false)
func (r *NewsSourceRepository) Delete(ctx context.Context, id int) error {
	query := `
		UPDATE news_sources 
		SET is_active = false, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1`
	
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete news source: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("news source with id %d not found", id)
	}
	
	r.logger.WithField("source_id", id).Info("Deleted news source")
	return nil
}

// GetByCountry возвращает источники новостей по стране
func (r *NewsSourceRepository) GetByCountry(ctx context.Context, countryID int) ([]models.NewsSource, error) {
	query := `
		SELECT id, name, domain, rss_url, website_url, country_id, language, 
			   description, logo_url, is_active, last_parsed_at, 
			   parse_interval_minutes, created_at, updated_at
		FROM news_sources
		WHERE country_id = $1 AND is_active = true
		ORDER BY name`
	
	rows, err := r.db.QueryContext(ctx, query, countryID)
	if err != nil {
		return nil, fmt.Errorf("failed to query news sources by country: %w", err)
	}
	defer rows.Close()
	
	var sources []models.NewsSource
	for rows.Next() {
		var source models.NewsSource
		err := rows.Scan(
			&source.ID, &source.Name, &source.Domain, &source.RSSURL,
			&source.WebsiteURL, &source.CountryID, &source.Language,
			&source.Description, &source.LogoURL, &source.IsActive,
			&source.LastParsedAt, &source.ParseIntervalMinutes,
			&source.CreatedAt, &source.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan news source: %w", err)
		}
		sources = append(sources, source)
	}
	
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}
	
	return sources, nil
}

// GetStats возвращает статистику по источникам новостей
func (r *NewsSourceRepository) GetStats(ctx context.Context) (*models.ParsingStats, error) {
	query := `
		SELECT 
			COUNT(*) as total_sources,
			COUNT(*) FILTER (WHERE is_active = true) as active_sources,
			MAX(last_parsed_at) as last_parse_time
		FROM news_sources`
	
	var stats models.ParsingStats
	var lastParseTime sql.NullTime
	
	err := r.db.QueryRowContext(ctx, query).Scan(
		&stats.TotalSources,
		&stats.ActiveSources,
		&lastParseTime,
	)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get sources stats: %w", err)
	}
	
	if lastParseTime.Valid {
		stats.LastParseTime = &lastParseTime.Time
	}
	
	return &stats, nil
}
