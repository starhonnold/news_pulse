package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/sirupsen/logrus"

	"news-management-service/internal/database"
	"news-management-service/internal/models"
)

// CategoryRepository представляет репозиторий для работы с категориями
type CategoryRepository struct {
	db     *database.DB
	logger *logrus.Logger
}

// NewCategoryRepository создает новый репозиторий категорий
func NewCategoryRepository(db *database.DB, logger *logrus.Logger) *CategoryRepository {
	return &CategoryRepository{
		db:     db,
		logger: logger,
	}
}

// GetAll возвращает все активные категории
func (r *CategoryRepository) GetAll(ctx context.Context) ([]models.Category, error) {
	query := `
		SELECT id, name, slug, color, icon, description, is_active, created_at, updated_at
		FROM categories
		WHERE is_active = true
		ORDER BY name`
	
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query categories: %w", err)
	}
	defer rows.Close()
	
	var categories []models.Category
	for rows.Next() {
		var category models.Category
		err := rows.Scan(
			&category.ID, &category.Name, &category.Slug, &category.Color,
			&category.Icon, &category.Description, &category.IsActive,
			&category.CreatedAt, &category.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}
		categories = append(categories, category)
	}
	
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}
	
	return categories, nil
}

// GetByID возвращает категорию по ID
func (r *CategoryRepository) GetByID(ctx context.Context, id int) (*models.Category, error) {
	query := `
		SELECT id, name, slug, color, icon, description, is_active, created_at, updated_at
		FROM categories
		WHERE id = $1 AND is_active = true`
	
	var category models.Category
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&category.ID, &category.Name, &category.Slug, &category.Color,
		&category.Icon, &category.Description, &category.IsActive,
		&category.CreatedAt, &category.UpdatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("category with id %d not found", id)
		}
		return nil, fmt.Errorf("failed to get category: %w", err)
	}
	
	return &category, nil
}

// GetBySlug возвращает категорию по slug
func (r *CategoryRepository) GetBySlug(ctx context.Context, slug string) (*models.Category, error) {
	query := `
		SELECT id, name, slug, color, icon, description, is_active, created_at, updated_at
		FROM categories
		WHERE slug = $1 AND is_active = true`
	
	var category models.Category
	err := r.db.QueryRowContext(ctx, query, slug).Scan(
		&category.ID, &category.Name, &category.Slug, &category.Color,
		&category.Icon, &category.Description, &category.IsActive,
		&category.CreatedAt, &category.UpdatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("category with slug '%s' not found", slug)
		}
		return nil, fmt.Errorf("failed to get category: %w", err)
	}
	
	return &category, nil
}

// GetWithNewsCount возвращает категории с количеством новостей
func (r *CategoryRepository) GetWithNewsCount(ctx context.Context) ([]models.CategoryStats, error) {
	query := `
		SELECT c.id, c.name, c.slug, c.color, c.icon, c.description, 
			   c.is_active, c.created_at, c.updated_at,
			   COUNT(n.id) as news_count
		FROM categories c
		LEFT JOIN news n ON c.id = n.category_id AND n.is_active = true
		WHERE c.is_active = true
		GROUP BY c.id, c.name, c.slug, c.color, c.icon, c.description, 
				 c.is_active, c.created_at, c.updated_at
		ORDER BY news_count DESC, c.name`
	
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query categories with news count: %w", err)
	}
	defer rows.Close()
	
	var categoryStats []models.CategoryStats
	for rows.Next() {
		var stat models.CategoryStats
		err := rows.Scan(
			&stat.Category.ID, &stat.Category.Name, &stat.Category.Slug,
			&stat.Category.Color, &stat.Category.Icon, &stat.Category.Description,
			&stat.Category.IsActive, &stat.Category.CreatedAt, &stat.Category.UpdatedAt,
			&stat.NewsCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan category stats: %w", err)
		}
		categoryStats = append(categoryStats, stat)
	}
	
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}
	
	return categoryStats, nil
}

// GetTopCategories возвращает топ категорий по количеству новостей
func (r *CategoryRepository) GetTopCategories(ctx context.Context, limit int) ([]models.CategoryStats, error) {
	query := `
		SELECT c.id, c.name, c.slug, c.color, c.icon, c.description, 
			   c.is_active, c.created_at, c.updated_at,
			   COUNT(n.id) as news_count
		FROM categories c
		LEFT JOIN news n ON c.id = n.category_id 
			AND n.is_active = true 
			AND n.published_at >= CURRENT_DATE - INTERVAL '7 days'
		WHERE c.is_active = true
		GROUP BY c.id, c.name, c.slug, c.color, c.icon, c.description, 
				 c.is_active, c.created_at, c.updated_at
		HAVING COUNT(n.id) > 0
		ORDER BY news_count DESC
		LIMIT $1`
	
	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query top categories: %w", err)
	}
	defer rows.Close()
	
	var categoryStats []models.CategoryStats
	for rows.Next() {
		var stat models.CategoryStats
		err := rows.Scan(
			&stat.Category.ID, &stat.Category.Name, &stat.Category.Slug,
			&stat.Category.Color, &stat.Category.Icon, &stat.Category.Description,
			&stat.Category.IsActive, &stat.Category.CreatedAt, &stat.Category.UpdatedAt,
			&stat.NewsCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan category stats: %w", err)
		}
		categoryStats = append(categoryStats, stat)
	}
	
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}
	
	return categoryStats, nil
}
