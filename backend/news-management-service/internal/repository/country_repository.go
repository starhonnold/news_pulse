package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/sirupsen/logrus"

	"news-management-service/internal/database"
	"news-management-service/internal/models"
)

// CountryRepository представляет репозиторий для работы со странами
type CountryRepository struct {
	db     *database.DB
	logger *logrus.Logger
}

// NewCountryRepository создает новый репозиторий стран
func NewCountryRepository(db *database.DB, logger *logrus.Logger) *CountryRepository {
	return &CountryRepository{
		db:     db,
		logger: logger,
	}
}

// GetAll возвращает все активные страны
func (r *CountryRepository) GetAll(ctx context.Context) ([]models.Country, error) {
	query := `
		SELECT id, name, code, flag_emoji, is_active, created_at, updated_at
		FROM countries
		WHERE is_active = true
		ORDER BY name`
	
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query countries: %w", err)
	}
	defer rows.Close()
	
	var countries []models.Country
	for rows.Next() {
		var country models.Country
		err := rows.Scan(
			&country.ID, &country.Name, &country.Code, &country.FlagEmoji,
			&country.IsActive, &country.CreatedAt, &country.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan country: %w", err)
		}
		countries = append(countries, country)
	}
	
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}
	
	return countries, nil
}

// GetByID возвращает страну по ID
func (r *CountryRepository) GetByID(ctx context.Context, id int) (*models.Country, error) {
	query := `
		SELECT id, name, code, flag_emoji, is_active, created_at, updated_at
		FROM countries
		WHERE id = $1 AND is_active = true`
	
	var country models.Country
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&country.ID, &country.Name, &country.Code, &country.FlagEmoji,
		&country.IsActive, &country.CreatedAt, &country.UpdatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("country with id %d not found", id)
		}
		return nil, fmt.Errorf("failed to get country: %w", err)
	}
	
	return &country, nil
}

// GetByCode возвращает страну по коду
func (r *CountryRepository) GetByCode(ctx context.Context, code string) (*models.Country, error) {
	query := `
		SELECT id, name, code, flag_emoji, is_active, created_at, updated_at
		FROM countries
		WHERE code = $1 AND is_active = true`
	
	var country models.Country
	err := r.db.QueryRowContext(ctx, query, code).Scan(
		&country.ID, &country.Name, &country.Code, &country.FlagEmoji,
		&country.IsActive, &country.CreatedAt, &country.UpdatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("country with code '%s' not found", code)
		}
		return nil, fmt.Errorf("failed to get country: %w", err)
	}
	
	return &country, nil
}

// GetWithSourcesCount возвращает страны с количеством источников
func (r *CountryRepository) GetWithSourcesCount(ctx context.Context) ([]models.CountryStats, error) {
	query := `
		SELECT c.id, c.name, c.code, c.flag_emoji, c.is_active, c.created_at, c.updated_at,
			   COUNT(ns.id) as news_count
		FROM countries c
		LEFT JOIN news_sources ns ON c.id = ns.country_id AND ns.is_active = true
		WHERE c.is_active = true
		GROUP BY c.id, c.name, c.code, c.flag_emoji, c.is_active, c.created_at, c.updated_at
		ORDER BY news_count DESC, c.name`
	
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query countries with sources count: %w", err)
	}
	defer rows.Close()
	
	var countryStats []models.CountryStats
	for rows.Next() {
		var stat models.CountryStats
		err := rows.Scan(
			&stat.Country.ID, &stat.Country.Name, &stat.Country.Code,
			&stat.Country.FlagEmoji, &stat.Country.IsActive,
			&stat.Country.CreatedAt, &stat.Country.UpdatedAt,
			&stat.NewsCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan country stats: %w", err)
		}
		countryStats = append(countryStats, stat)
	}
	
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}
	
	return countryStats, nil
}

// GetWithNewsCount возвращает страны с количеством новостей
func (r *CountryRepository) GetWithNewsCount(ctx context.Context) ([]models.CountryStats, error) {
	query := `
		SELECT c.id, c.name, c.code, c.flag_emoji, c.is_active, c.created_at, c.updated_at,
			   COUNT(n.id) as news_count
		FROM countries c
		LEFT JOIN news_sources ns ON c.id = ns.country_id AND ns.is_active = true
		LEFT JOIN news n ON ns.id = n.source_id AND n.is_active = true
		WHERE c.is_active = true
		GROUP BY c.id, c.name, c.code, c.flag_emoji, c.is_active, c.created_at, c.updated_at
		ORDER BY news_count DESC, c.name`
	
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query countries with news count: %w", err)
	}
	defer rows.Close()
	
	var countryStats []models.CountryStats
	for rows.Next() {
		var stat models.CountryStats
		err := rows.Scan(
			&stat.Country.ID, &stat.Country.Name, &stat.Country.Code,
			&stat.Country.FlagEmoji, &stat.Country.IsActive,
			&stat.Country.CreatedAt, &stat.Country.UpdatedAt,
			&stat.NewsCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan country stats: %w", err)
		}
		countryStats = append(countryStats, stat)
	}
	
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}
	
	return countryStats, nil
}

// GetTopCountries возвращает топ стран по количеству новостей
func (r *CountryRepository) GetTopCountries(ctx context.Context, limit int) ([]models.CountryStats, error) {
	query := `
		SELECT c.id, c.name, c.code, c.flag_emoji, c.is_active, c.created_at, c.updated_at,
			   COUNT(n.id) as news_count
		FROM countries c
		LEFT JOIN news_sources ns ON c.id = ns.country_id AND ns.is_active = true
		LEFT JOIN news n ON ns.id = n.source_id 
			AND n.is_active = true 
			AND n.published_at >= CURRENT_DATE - INTERVAL '7 days'
		WHERE c.is_active = true
		GROUP BY c.id, c.name, c.code, c.flag_emoji, c.is_active, c.created_at, c.updated_at
		HAVING COUNT(n.id) > 0
		ORDER BY news_count DESC
		LIMIT $1`
	
	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query top countries: %w", err)
	}
	defer rows.Close()
	
	var countryStats []models.CountryStats
	for rows.Next() {
		var stat models.CountryStats
		err := rows.Scan(
			&stat.Country.ID, &stat.Country.Name, &stat.Country.Code,
			&stat.Country.FlagEmoji, &stat.Country.IsActive,
			&stat.Country.CreatedAt, &stat.Country.UpdatedAt,
			&stat.NewsCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan country stats: %w", err)
		}
		countryStats = append(countryStats, stat)
	}
	
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}
	
	return countryStats, nil
}
