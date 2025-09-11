package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"

	"news-parsing-service/internal/database"
	"news-parsing-service/internal/models"
)

// NewsRepository представляет репозиторий для работы с новостями
type NewsRepository struct {
	db     *database.DB
	logger *logrus.Logger
}

// NewNewsRepository создает новый репозиторий новостей
func NewNewsRepository(db *database.DB, logger *logrus.Logger) *NewsRepository {
	return &NewsRepository{
		db:     db,
		logger: logger,
	}
}

// Create создает новую новость
func (r *NewsRepository) Create(ctx context.Context, news *models.News) error {
	query := `
		INSERT INTO news (title, description, content, url, image_url, author, 
						 source_id, category_id, published_at, relevance_score)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, parsed_at, created_at, updated_at`
	
	err := r.db.QueryRowContext(ctx, query,
		news.Title, news.Description, news.Content, news.URL,
		news.ImageURL, news.Author, news.SourceID, news.CategoryID,
		news.PublishedAt, news.RelevanceScore,
	).Scan(&news.ID, &news.ParsedAt, &news.CreatedAt, &news.UpdatedAt)
	
	if err != nil {
		return fmt.Errorf("failed to create news: %w", err)
	}
	
	r.logger.WithFields(logrus.Fields{
		"news_id":   news.ID,
		"source_id": news.SourceID,
		"title":     truncateString(news.Title, 50),
	}).Debug("Created news")
	
	return nil
}

// CreateBatch создает новости пакетом для оптимизации производительности
func (r *NewsRepository) CreateBatch(ctx context.Context, newsList []models.News) error {
	if len(newsList) == 0 {
		return nil
	}
	
	// Используем транзакцию для атомарности
	return r.db.Transaction(func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO news (title, description, content, url, image_url, author, 
							 source_id, category_id, published_at, relevance_score)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			ON CONFLICT (url, source_id) DO NOTHING
			RETURNING id`)
		
		if err != nil {
			return fmt.Errorf("failed to prepare statement: %w", err)
		}
		defer stmt.Close()
		
		successCount := 0
		for i := range newsList {
			news := &newsList[i]
			var id int
			err := stmt.QueryRowContext(ctx,
				news.Title, news.Description, news.Content, news.URL,
				news.ImageURL, news.Author, news.SourceID, news.CategoryID,
				news.PublishedAt, news.RelevanceScore,
			).Scan(&id)
			
			if err != nil {
				if err == sql.ErrNoRows {
					// Новость уже существует, пропускаем
					r.logger.WithFields(logrus.Fields{
						"url":       news.URL,
						"source_id": news.SourceID,
					}).Debug("News already exists, skipping")
					continue
				}
				return fmt.Errorf("failed to insert news: %w", err)
			}
			
			news.ID = id
			successCount++
		}
		
		r.logger.WithFields(logrus.Fields{
			"total":   len(newsList),
			"created": successCount,
			"skipped": len(newsList) - successCount,
		}).Info("Batch created news")
		
		return nil
	})
}

// GetByID возвращает новость по ID
func (r *NewsRepository) GetByID(ctx context.Context, id int) (*models.News, error) {
	query := `
		SELECT n.id, n.title, n.description, n.content, n.url, n.image_url, 
			   n.author, n.source_id, n.category_id, n.published_at, n.parsed_at,
			   n.relevance_score, n.view_count, n.is_active, n.created_at, n.updated_at,
			   ns.name as source_name, ns.domain as source_domain,
			   c.name as category_name, c.slug as category_slug, c.color as category_color
		FROM news n
		JOIN news_sources ns ON n.source_id = ns.id
		LEFT JOIN categories c ON n.category_id = c.id
		WHERE n.id = $1`
	
	var news models.News
	var sourceName, sourceDomain string
	var categoryName, categorySlug, categoryColor sql.NullString
	
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&news.ID, &news.Title, &news.Description, &news.Content,
		&news.URL, &news.ImageURL, &news.Author, &news.SourceID,
		&news.CategoryID, &news.PublishedAt, &news.ParsedAt,
		&news.RelevanceScore, &news.ViewCount, &news.IsActive,
		&news.CreatedAt, &news.UpdatedAt,
		&sourceName, &sourceDomain,
		&categoryName, &categorySlug, &categoryColor,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("news with id %d not found", id)
		}
		return nil, fmt.Errorf("failed to get news: %w", err)
	}
	
	// Заполняем связанные данные
	news.Source = &models.NewsSource{
		ID:     news.SourceID,
		Name:   sourceName,
		Domain: sourceDomain,
	}
	
	if categoryName.Valid {
		news.Category = &models.Category{
			ID:    *news.CategoryID,
			Name:  categoryName.String,
			Slug:  categorySlug.String,
			Color: categoryColor.String,
		}
	}
	
	return &news, nil
}

// GetByFilter возвращает новости по фильтру
func (r *NewsRepository) GetByFilter(ctx context.Context, filter models.NewsFilter) ([]models.News, error) {
	query, args := r.buildFilterQuery(filter)
	
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query news: %w", err)
	}
	defer rows.Close()
	
	var newsList []models.News
	for rows.Next() {
		var news models.News
		var sourceName, sourceDomain string
		var categoryName, categorySlug, categoryColor sql.NullString
		
		err := rows.Scan(
			&news.ID, &news.Title, &news.Description, &news.Content,
			&news.URL, &news.ImageURL, &news.Author, &news.SourceID,
			&news.CategoryID, &news.PublishedAt, &news.ParsedAt,
			&news.RelevanceScore, &news.ViewCount, &news.IsActive,
			&news.CreatedAt, &news.UpdatedAt,
			&sourceName, &sourceDomain,
			&categoryName, &categorySlug, &categoryColor,
		)
		
		if err != nil {
			return nil, fmt.Errorf("failed to scan news: %w", err)
		}
		
		// Заполняем связанные данные
		news.Source = &models.NewsSource{
			ID:     news.SourceID,
			Name:   sourceName,
			Domain: sourceDomain,
		}
		
		if categoryName.Valid {
			news.Category = &models.Category{
				ID:    *news.CategoryID,
				Name:  categoryName.String,
				Slug:  categorySlug.String,
				Color: categoryColor.String,
			}
		}
		
		newsList = append(newsList, news)
	}
	
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}
	
	return newsList, nil
}

// buildFilterQuery строит SQL запрос на основе фильтра
func (r *NewsRepository) buildFilterQuery(filter models.NewsFilter) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	argIndex := 1
	
	baseQuery := `
		SELECT n.id, n.title, n.description, n.content, n.url, n.image_url, 
			   n.author, n.source_id, n.category_id, n.published_at, n.parsed_at,
			   n.relevance_score, n.view_count, n.is_active, n.created_at, n.updated_at,
			   ns.name as source_name, ns.domain as source_domain,
			   c.name as category_name, c.slug as category_slug, c.color as category_color
		FROM news n
		JOIN news_sources ns ON n.source_id = ns.id
		LEFT JOIN categories c ON n.category_id = c.id`
	
	// Фильтр по источникам
	if len(filter.SourceIDs) > 0 {
		placeholders := make([]string, len(filter.SourceIDs))
		for i, sourceID := range filter.SourceIDs {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, sourceID)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("n.source_id IN (%s)", strings.Join(placeholders, ",")))
	}
	
	// Фильтр по категориям
	if len(filter.CategoryIDs) > 0 {
		placeholders := make([]string, len(filter.CategoryIDs))
		for i, categoryID := range filter.CategoryIDs {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, categoryID)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("n.category_id IN (%s)", strings.Join(placeholders, ",")))
	}
	
	// Фильтр по странам (через источники)
	if len(filter.CountryIDs) > 0 {
		placeholders := make([]string, len(filter.CountryIDs))
		for i, countryID := range filter.CountryIDs {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, countryID)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("ns.country_id IN (%s)", strings.Join(placeholders, ",")))
	}
	
	// Фильтр по ключевым словам (полнотекстовый поиск)
	if filter.Keywords != "" {
		conditions = append(conditions, 
			fmt.Sprintf("to_tsvector('russian', n.title || ' ' || COALESCE(n.description, '')) @@ plainto_tsquery('russian', $%d)", argIndex))
		args = append(args, filter.Keywords)
		argIndex++
	}
	
	// Фильтр по дате от
	if filter.DateFrom != nil {
		conditions = append(conditions, fmt.Sprintf("n.published_at >= $%d", argIndex))
		args = append(args, *filter.DateFrom)
		argIndex++
	}
	
	// Фильтр по дате до
	if filter.DateTo != nil {
		conditions = append(conditions, fmt.Sprintf("n.published_at <= $%d", argIndex))
		args = append(args, *filter.DateTo)
		argIndex++
	}
	
	// Фильтр по активности
	if filter.IsActive != nil {
		conditions = append(conditions, fmt.Sprintf("n.is_active = $%d", argIndex))
		args = append(args, *filter.IsActive)
		argIndex++
	}
	
	// Добавляем условия WHERE
	if len(conditions) > 0 {
		baseQuery += " WHERE " + strings.Join(conditions, " AND ")
	}
	
	// Сортировка
	sortBy := filter.SortBy
	if sortBy == "" {
		sortBy = models.SortByPublishedAt
	}
	
	sortOrder := filter.SortOrder
	if sortOrder == "" {
		sortOrder = models.SortOrderDesc
	}
	
	baseQuery += fmt.Sprintf(" ORDER BY n.%s %s", sortBy, strings.ToUpper(sortOrder))
	
	// Лимит и оффсет
	if filter.Limit > 0 {
		baseQuery += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filter.Limit)
		argIndex++
	}
	
	if filter.Offset > 0 {
		baseQuery += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, filter.Offset)
		argIndex++
	}
	
	return baseQuery, args
}

// ExistsByURL проверяет существование новости по URL и источнику
func (r *NewsRepository) ExistsByURL(ctx context.Context, url string, sourceID int) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM news WHERE url = $1 AND source_id = $2)`
	
	var exists bool
	err := r.db.QueryRowContext(ctx, query, url, sourceID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check news existence: %w", err)
	}
	
	return exists, nil
}

// UpdateViewCount увеличивает счетчик просмотров новости
func (r *NewsRepository) UpdateViewCount(ctx context.Context, newsID int) error {
	query := `UPDATE news SET view_count = view_count + 1, updated_at = CURRENT_TIMESTAMP WHERE id = $1`
	
	result, err := r.db.ExecContext(ctx, query, newsID)
	if err != nil {
		return fmt.Errorf("failed to update view count: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("news with id %d not found", newsID)
	}
	
	return nil
}

// GetLatest возвращает последние новости
func (r *NewsRepository) GetLatest(ctx context.Context, limit int) ([]models.News, error) {
	filter := models.NewsFilter{
		Limit:     limit,
		SortBy:    models.SortByPublishedAt,
		SortOrder: models.SortOrderDesc,
		IsActive:  &[]bool{true}[0],
	}
	
	return r.GetByFilter(ctx, filter)
}

// GetBySource возвращает новости по источнику
func (r *NewsRepository) GetBySource(ctx context.Context, sourceID int, limit int) ([]models.News, error) {
	filter := models.NewsFilter{
		SourceIDs: []int{sourceID},
		Limit:     limit,
		SortBy:    models.SortByPublishedAt,
		SortOrder: models.SortOrderDesc,
		IsActive:  &[]bool{true}[0],
	}
	
	return r.GetByFilter(ctx, filter)
}

// GetStats возвращает статистику по новостям
func (r *NewsRepository) GetNewsStats(ctx context.Context) (*models.ParsingStats, error) {
	query := `
		SELECT 
			COUNT(*) as total_news,
			COUNT(*) FILTER (WHERE DATE(created_at) = CURRENT_DATE) as news_today
		FROM news
		WHERE is_active = true`
	
	var totalNews, newsToday int
	err := r.db.QueryRowContext(ctx, query).Scan(&totalNews, &newsToday)
	if err != nil {
		return nil, fmt.Errorf("failed to get news stats: %w", err)
	}
	
	return &models.ParsingStats{
		TotalNews: totalNews,
		NewsToday: newsToday,
	}, nil
}

// CleanupOldNews удаляет старые новости (мягкое удаление)
func (r *NewsRepository) CleanupOldNews(ctx context.Context, retentionDays int) (int, error) {
	query := `
		UPDATE news 
		SET is_active = false, updated_at = CURRENT_TIMESTAMP
		WHERE published_at < NOW() - INTERVAL '%d days'
		  AND is_active = true`
	
	result, err := r.db.ExecContext(ctx, fmt.Sprintf(query, retentionDays))
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup old news: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	r.logger.WithFields(logrus.Fields{
		"retention_days": retentionDays,
		"cleaned_count":  rowsAffected,
	}).Info("Cleaned up old news")
	
	return int(rowsAffected), nil
}

// Вспомогательная функция для обрезания строки
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
