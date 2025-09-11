package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"news-management-service/internal/database"
	"news-management-service/internal/models"
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

// GetByID возвращает новость по ID
func (r *NewsRepository) GetByID(ctx context.Context, id int) (*models.News, error) {
	query := `
		SELECT n.id, n.title, n.description, n.content, n.url, n.image_url, 
			   n.author, n.source_id, n.category_id, n.published_at, n.parsed_at,
			   n.relevance_score, n.view_count, n.is_active, n.created_at, n.updated_at,
			   ns.name as source_name, ns.domain as source_domain, ns.logo_url as source_logo,
			   c.name as category_name, c.slug as category_slug, c.color as category_color, c.icon as category_icon,
			   co.name as country_name, co.code as country_code, co.flag_emoji as country_flag
		FROM news n
		JOIN news_sources ns ON n.source_id = ns.id
		LEFT JOIN categories c ON n.category_id = c.id
		LEFT JOIN countries co ON ns.country_id = co.id
		WHERE n.id = $1 AND n.is_active = true`
	
	var news models.News
	var sourceName, sourceDomain, sourceLogoURL string
	var categoryName, categorySlug, categoryColor, categoryIcon sql.NullString
	var countryName, countryCode, countryFlag sql.NullString
	
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&news.ID, &news.Title, &news.Description, &news.Content,
		&news.URL, &news.ImageURL, &news.Author, &news.SourceID,
		&news.CategoryID, &news.PublishedAt, &news.ParsedAt,
		&news.RelevanceScore, &news.ViewCount, &news.IsActive,
		&news.CreatedAt, &news.UpdatedAt,
		&sourceName, &sourceDomain, &sourceLogoURL,
		&categoryName, &categorySlug, &categoryColor, &categoryIcon,
		&countryName, &countryCode, &countryFlag,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("news with id %d not found", id)
		}
		return nil, fmt.Errorf("failed to get news: %w", err)
	}
	
	// Заполняем связанные данные
	news.Source = &models.NewsSource{
		ID:      news.SourceID,
		Name:    sourceName,
		Domain:  sourceDomain,
		LogoURL: sourceLogoURL,
	}
	
	if categoryName.Valid {
		news.Category = &models.Category{
			ID:    *news.CategoryID,
			Name:  categoryName.String,
			Slug:  categorySlug.String,
			Color: categoryColor.String,
			Icon:  categoryIcon.String,
		}
	}
	
	if countryName.Valid {
		news.Country = &models.Country{
			Name:      countryName.String,
			Code:      countryCode.String,
			FlagEmoji: countryFlag.String,
		}
	}
	
	// Получаем теги для новости
	tags, err := r.getNewsTags(ctx, news.ID)
	if err != nil {
		r.logger.WithError(err).Warn("Failed to get news tags")
	} else {
		news.Tags = tags
	}
	
	return &news, nil
}

// GetByFilter возвращает новости по фильтру с пагинацией
func (r *NewsRepository) GetByFilter(ctx context.Context, filter models.NewsFilter) (*models.NewsResponse, error) {
	// Строим запрос для подсчета общего количества
	countQuery, countArgs := r.buildCountQuery(filter)
	
	var total int
	err := r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count news: %w", err)
	}
	
	// Строим основной запрос
	query, args := r.buildFilterQuery(filter)
	
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query news: %w", err)
	}
	defer rows.Close()
	
	var newsList []models.News
	for rows.Next() {
		var news models.News
		var sourceName, sourceDomain, sourceLogoURL string
		var categoryName, categorySlug, categoryColor, categoryIcon sql.NullString
		var countryName, countryCode, countryFlag sql.NullString
		
		err := rows.Scan(
			&news.ID, &news.Title, &news.Description, &news.Content,
			&news.URL, &news.ImageURL, &news.Author, &news.SourceID,
			&news.CategoryID, &news.PublishedAt, &news.ParsedAt,
			&news.RelevanceScore, &news.ViewCount, &news.IsActive,
			&news.CreatedAt, &news.UpdatedAt,
			&sourceName, &sourceDomain, &sourceLogoURL,
			&categoryName, &categorySlug, &categoryColor, &categoryIcon,
			&countryName, &countryCode, &countryFlag,
		)
		
		if err != nil {
			return nil, fmt.Errorf("failed to scan news: %w", err)
		}
		
		// Заполняем связанные данные
		news.Source = &models.NewsSource{
			ID:      news.SourceID,
			Name:    sourceName,
			Domain:  sourceDomain,
			LogoURL: sourceLogoURL,
		}
		
		if categoryName.Valid {
			news.Category = &models.Category{
				ID:    *news.CategoryID,
				Name:  categoryName.String,
				Slug:  categorySlug.String,
				Color: categoryColor.String,
				Icon:  categoryIcon.String,
			}
		}
		
		if countryName.Valid {
			news.Country = &models.Country{
				Name:      countryName.String,
				Code:      countryCode.String,
				FlagEmoji: countryFlag.String,
			}
		}
		
		newsList = append(newsList, news)
	}
	
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}
	
	// Создаем пагинацию
	pagination := models.NewPagination(filter.Page, filter.PageSize, total)
	
	return &models.NewsResponse{
		News:       newsList,
		Pagination: pagination,
		Filters:    filter,
	}, nil
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
			   ns.name as source_name, ns.domain as source_domain, ns.logo_url as source_logo,
			   c.name as category_name, c.slug as category_slug, c.color as category_color, c.icon as category_icon,
			   co.name as country_name, co.code as country_code, co.flag_emoji as country_flag
		FROM news n
		JOIN news_sources ns ON n.source_id = ns.id
		LEFT JOIN categories c ON n.category_id = c.id
		LEFT JOIN countries co ON ns.country_id = co.id`
	
	// Базовое условие - только активные новости
	conditions = append(conditions, "n.is_active = true")
	
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
	
	// Фильтр по минимальной релевантности
	if filter.MinRelevance != nil {
		conditions = append(conditions, fmt.Sprintf("n.relevance_score >= $%d", argIndex))
		args = append(args, *filter.MinRelevance)
		argIndex++
	}
	
	// Добавляем условия WHERE
	if len(conditions) > 0 {
		baseQuery += " WHERE " + strings.Join(conditions, " AND ")
	}
	
	// Сортировка
	sortBy, sortOrder := models.NormalizeSortParams(filter.SortBy, filter.SortOrder)
	baseQuery += fmt.Sprintf(" ORDER BY n.%s %s", sortBy, strings.ToUpper(sortOrder))
	
	// Лимит и оффсет
	baseQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, filter.PageSize, filter.GetOffset())
	
	return baseQuery, args
}

// buildCountQuery строит запрос для подсчета общего количества
func (r *NewsRepository) buildCountQuery(filter models.NewsFilter) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	argIndex := 1
	
	baseQuery := `
		SELECT COUNT(*)
		FROM news n
		JOIN news_sources ns ON n.source_id = ns.id`
	
	// Базовое условие - только активные новости
	conditions = append(conditions, "n.is_active = true")
	
	// Те же фильтры, что и в основном запросе
	if len(filter.SourceIDs) > 0 {
		placeholders := make([]string, len(filter.SourceIDs))
		for i, sourceID := range filter.SourceIDs {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, sourceID)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("n.source_id IN (%s)", strings.Join(placeholders, ",")))
	}
	
	if len(filter.CategoryIDs) > 0 {
		placeholders := make([]string, len(filter.CategoryIDs))
		for i, categoryID := range filter.CategoryIDs {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, categoryID)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("n.category_id IN (%s)", strings.Join(placeholders, ",")))
	}
	
	if len(filter.CountryIDs) > 0 {
		placeholders := make([]string, len(filter.CountryIDs))
		for i, countryID := range filter.CountryIDs {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, countryID)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("ns.country_id IN (%s)", strings.Join(placeholders, ",")))
	}
	
	if filter.Keywords != "" {
		conditions = append(conditions, 
			fmt.Sprintf("to_tsvector('russian', n.title || ' ' || COALESCE(n.description, '')) @@ plainto_tsquery('russian', $%d)", argIndex))
		args = append(args, filter.Keywords)
		argIndex++
	}
	
	if filter.DateFrom != nil {
		conditions = append(conditions, fmt.Sprintf("n.published_at >= $%d", argIndex))
		args = append(args, *filter.DateFrom)
		argIndex++
	}
	
	if filter.DateTo != nil {
		conditions = append(conditions, fmt.Sprintf("n.published_at <= $%d", argIndex))
		args = append(args, *filter.DateTo)
		argIndex++
	}
	
	if filter.MinRelevance != nil {
		conditions = append(conditions, fmt.Sprintf("n.relevance_score >= $%d", argIndex))
		args = append(args, *filter.MinRelevance)
		argIndex++
	}
	
	// Добавляем условия WHERE
	if len(conditions) > 0 {
		baseQuery += " WHERE " + strings.Join(conditions, " AND ")
	}
	
	return baseQuery, args
}

// getNewsTags получает теги для новости
func (r *NewsRepository) getNewsTags(ctx context.Context, newsID int) ([]models.Tag, error) {
	query := `
		SELECT t.id, t.name, t.slug, t.usage_count, t.created_at
		FROM tags t
		JOIN news_tags nt ON t.id = nt.tag_id
		WHERE nt.news_id = $1
		ORDER BY t.name`
	
	rows, err := r.db.QueryContext(ctx, query, newsID)
	if err != nil {
		return nil, fmt.Errorf("failed to query news tags: %w", err)
	}
	defer rows.Close()
	
	var tags []models.Tag
	for rows.Next() {
		var tag models.Tag
		err := rows.Scan(&tag.ID, &tag.Name, &tag.Slug, &tag.UsageCount, &tag.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tag: %w", err)
		}
		tags = append(tags, tag)
	}
	
	return tags, rows.Err()
}

// UpdateViewCount увеличивает счетчик просмотров новости
func (r *NewsRepository) UpdateViewCount(ctx context.Context, newsID int) error {
	query := `UPDATE news SET view_count = view_count + 1, updated_at = CURRENT_TIMESTAMP WHERE id = $1 AND is_active = true`
	
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
	
	r.logger.WithField("news_id", newsID).Debug("Updated view count")
	return nil
}

// GetLatest возвращает последние новости
func (r *NewsRepository) GetLatest(ctx context.Context, limit int) ([]models.News, error) {
	filter := models.NewsFilter{
		Page:      1,
		PageSize:  limit,
		SortBy:    models.SortByPublishedAt,
		SortOrder: models.SortOrderDesc,
	}
	
	response, err := r.GetByFilter(ctx, filter)
	if err != nil {
		return nil, err
	}
	
	return response.News, nil
}

// GetTrending возвращает трендовые новости
func (r *NewsRepository) GetTrending(ctx context.Context, limit int) ([]models.News, error) {
	// Трендовые новости - с высокой релевантностью и просмотрами за последние 24 часа
	since := time.Now().Add(-24 * time.Hour)
	
	filter := models.NewsFilter{
		Page:         1,
		PageSize:     limit,
		DateFrom:     &since,
		MinRelevance: &[]float64{0.7}[0],
		SortBy:       models.SortByViewCount,
		SortOrder:    models.SortOrderDesc,
	}
	
	response, err := r.GetByFilter(ctx, filter)
	if err != nil {
		return nil, err
	}
	
	return response.News, nil
}

// Search выполняет полнотекстовый поиск новостей
func (r *NewsRepository) Search(ctx context.Context, query string, page, pageSize int) (*models.SearchResult, error) {
	start := time.Now()
	
	filter := models.NewsFilter{
		Keywords:  query,
		Page:      page,
		PageSize:  pageSize,
		SortBy:    models.SortByRelevanceScore,
		SortOrder: models.SortOrderDesc,
	}
	
	response, err := r.GetByFilter(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to search news: %w", err)
	}
	
	searchTime := time.Since(start)
	
	return &models.SearchResult{
		News:       response.News,
		Pagination: response.Pagination,
		Query:      query,
		SearchTime: fmt.Sprintf("%.2fms", float64(searchTime.Nanoseconds())/1000000),
	}, nil
}
