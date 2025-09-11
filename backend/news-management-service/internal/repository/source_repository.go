package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/sirupsen/logrus"

	"news-management-service/internal/database"
	"news-management-service/internal/models"
)

// SourceRepository представляет репозиторий для работы с источниками новостей
type SourceRepository struct {
	db     *database.DB
	logger *logrus.Logger
}

// NewSourceRepository создает новый репозиторий источников
func NewSourceRepository(db *database.DB, logger *logrus.Logger) *SourceRepository {
	return &SourceRepository{
		db:     db,
		logger: logger,
	}
}

// GetAll возвращает все активные источники новостей
func (r *SourceRepository) GetAll(ctx context.Context) ([]models.NewsSource, error) {
	query := `
		SELECT ns.id, ns.name, ns.domain, ns.rss_url, ns.website_url, ns.country_id, 
			   ns.language, ns.description, ns.logo_url, ns.is_active, ns.last_parsed_at, 
			   ns.parse_interval_minutes, ns.created_at, ns.updated_at
		FROM news_sources ns
		WHERE ns.is_active = true
		ORDER BY ns.name`
	
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

// GetByID возвращает источник новостей по ID
func (r *SourceRepository) GetByID(ctx context.Context, id int) (*models.NewsSource, error) {
	query := `
		SELECT ns.id, ns.name, ns.domain, ns.rss_url, ns.website_url, ns.country_id, 
			   ns.language, ns.description, ns.logo_url, ns.is_active, ns.last_parsed_at, 
			   ns.parse_interval_minutes, ns.created_at, ns.updated_at
		FROM news_sources ns
		WHERE ns.id = $1 AND ns.is_active = true`
	
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

// GetByCountry возвращает источники новостей по стране
func (r *SourceRepository) GetByCountry(ctx context.Context, countryID int) ([]models.NewsSource, error) {
	query := `
		SELECT ns.id, ns.name, ns.domain, ns.rss_url, ns.website_url, ns.country_id, 
			   ns.language, ns.description, ns.logo_url, ns.is_active, ns.last_parsed_at, 
			   ns.parse_interval_minutes, ns.created_at, ns.updated_at
		FROM news_sources ns
		WHERE ns.country_id = $1 AND ns.is_active = true
		ORDER BY ns.name`
	
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

// GetWithNewsCount возвращает источники с количеством новостей
func (r *SourceRepository) GetWithNewsCount(ctx context.Context) ([]models.SourceStats, error) {
	query := `
		SELECT ns.id, ns.name, ns.domain, ns.rss_url, ns.website_url, ns.country_id, 
			   ns.language, ns.description, ns.logo_url, ns.is_active, ns.last_parsed_at, 
			   ns.parse_interval_minutes, ns.created_at, ns.updated_at,
			   COUNT(n.id) as news_count
		FROM news_sources ns
		LEFT JOIN news n ON ns.id = n.source_id AND n.is_active = true
		WHERE ns.is_active = true
		GROUP BY ns.id, ns.name, ns.domain, ns.rss_url, ns.website_url, ns.country_id, 
				 ns.language, ns.description, ns.logo_url, ns.is_active, ns.last_parsed_at, 
				 ns.parse_interval_minutes, ns.created_at, ns.updated_at
		ORDER BY news_count DESC, ns.name`
	
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query sources with news count: %w", err)
	}
	defer rows.Close()
	
	var sourceStats []models.SourceStats
	for rows.Next() {
		var stat models.SourceStats
		err := rows.Scan(
			&stat.Source.ID, &stat.Source.Name, &stat.Source.Domain, &stat.Source.RSSURL,
			&stat.Source.WebsiteURL, &stat.Source.CountryID, &stat.Source.Language,
			&stat.Source.Description, &stat.Source.LogoURL, &stat.Source.IsActive,
			&stat.Source.LastParsedAt, &stat.Source.ParseIntervalMinutes,
			&stat.Source.CreatedAt, &stat.Source.UpdatedAt,
			&stat.NewsCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan source stats: %w", err)
		}
		sourceStats = append(sourceStats, stat)
	}
	
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}
	
	return sourceStats, nil
}

// GetTopSources возвращает топ источников по количеству новостей
func (r *SourceRepository) GetTopSources(ctx context.Context, limit int) ([]models.SourceStats, error) {
	query := `
		SELECT ns.id, ns.name, ns.domain, ns.rss_url, ns.website_url, ns.country_id, 
			   ns.language, ns.description, ns.logo_url, ns.is_active, ns.last_parsed_at, 
			   ns.parse_interval_minutes, ns.created_at, ns.updated_at,
			   COUNT(n.id) as news_count
		FROM news_sources ns
		LEFT JOIN news n ON ns.id = n.source_id 
			AND n.is_active = true 
			AND n.published_at >= CURRENT_DATE - INTERVAL '7 days'
		WHERE ns.is_active = true
		GROUP BY ns.id, ns.name, ns.domain, ns.rss_url, ns.website_url, ns.country_id, 
				 ns.language, ns.description, ns.logo_url, ns.is_active, ns.last_parsed_at, 
				 ns.parse_interval_minutes, ns.created_at, ns.updated_at
		HAVING COUNT(n.id) > 0
		ORDER BY news_count DESC
		LIMIT $1`
	
	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query top sources: %w", err)
	}
	defer rows.Close()
	
	var sourceStats []models.SourceStats
	for rows.Next() {
		var stat models.SourceStats
		err := rows.Scan(
			&stat.Source.ID, &stat.Source.Name, &stat.Source.Domain, &stat.Source.RSSURL,
			&stat.Source.WebsiteURL, &stat.Source.CountryID, &stat.Source.Language,
			&stat.Source.Description, &stat.Source.LogoURL, &stat.Source.IsActive,
			&stat.Source.LastParsedAt, &stat.Source.ParseIntervalMinutes,
			&stat.Source.CreatedAt, &stat.Source.UpdatedAt,
			&stat.NewsCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan source stats: %w", err)
		}
		sourceStats = append(sourceStats, stat)
	}
	
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}
	
	return sourceStats, nil
}
