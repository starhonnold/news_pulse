package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"pulse-service/internal/database"
	"pulse-service/internal/models"
)

// FeedRepository представляет репозиторий для работы с персонализированными лентами
type FeedRepository struct {
	db     *database.DB
	logger *logrus.Logger
}

// NewFeedRepository создает новый репозиторий лент
func NewFeedRepository(db *database.DB, logger *logrus.Logger) *FeedRepository {
	return &FeedRepository{
		db:     db,
		logger: logger,
	}
}

// GetPersonalizedFeed возвращает персонализированную ленту новостей для пульса
func (r *FeedRepository) GetPersonalizedFeed(ctx context.Context, pulse *models.UserPulse, req models.FeedRequest) (*models.PersonalizedFeed, error) {
	// Строим запрос для подсчета общего количества
	countQuery, countArgs := r.buildFeedCountQuery(pulse, req)
	
	var total int
	err := r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count personalized feed: %w", err)
	}
	
	// Строим основной запрос
	query, args := r.buildFeedQuery(pulse, req)
	
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query personalized feed: %w", err)
	}
	defer rows.Close()
	
	var newsList []models.PersonalizedNews
	for rows.Next() {
		var news models.PersonalizedNews
		var categoryName, categorySlug, categoryColor sql.NullString
		
		err := rows.Scan(
			&news.ID, &news.Title, &news.Description, &news.URL,
			&news.ImageURL, &news.Author, &news.SourceID, &news.CategoryID,
			&news.PublishedAt, &news.RelevanceScore, &news.ViewCount,
			&news.SourceName, &news.SourceDomain, &news.SourceLogoURL,
			&categoryName, &categorySlug, &categoryColor,
		)
		
		if err != nil {
			return nil, fmt.Errorf("failed to scan personalized news: %w", err)
		}
		
		// Заполняем категорию если есть
		if categoryName.Valid {
			news.CategoryName = categoryName.String
			news.CategorySlug = categorySlug.String
			news.CategoryColor = categoryColor.String
		}
		
		// Определяем причину попадания новости в ленту
		news.MatchReason = r.determineMatchReason(pulse, news)
		
		// Вычисляем персональный скор
		news.CalculatePersonalScore()
		
		newsList = append(newsList, news)
	}
	
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}
	
	// Создаем пагинацию
	pagination := models.NewPagination(req.Page, req.PageSize, total)
	
	// Собираем статистику
	stats := r.calculateFeedStats(newsList)
	
	return &models.PersonalizedFeed{
		PulseID:     pulse.ID,
		PulseName:   pulse.Name,
		News:        newsList,
		Pagination:  pagination,
		GeneratedAt: time.Now(),
		Stats:       stats,
	}, nil
}

// GetFeedByDateRange возвращает новости пульса за определенный период
func (r *FeedRepository) GetFeedByDateRange(ctx context.Context, pulse *models.UserPulse, from, to time.Time, limit int) ([]models.PersonalizedNews, error) {
	req := models.FeedRequest{
		PulseID:   pulse.ID,
		Page:      1,
		PageSize:  limit,
		DateFrom:  &from,
		DateTo:    &to,
		SortBy:    models.FeedSortByPublishedAt,
		SortOrder: models.SortOrderDesc,
	}
	
	feed, err := r.GetPersonalizedFeed(ctx, pulse, req)
	if err != nil {
		return nil, err
	}
	
	return feed.News, nil
}

// GetLatestFeedNews возвращает последние новости для пульса
func (r *FeedRepository) GetLatestFeedNews(ctx context.Context, pulse *models.UserPulse, limit int) ([]models.PersonalizedNews, error) {
	req := models.FeedRequest{
		PulseID:   pulse.ID,
		Page:      1,
		PageSize:  limit,
		SortBy:    models.FeedSortByPublishedAt,
		SortOrder: models.SortOrderDesc,
	}
	
	feed, err := r.GetPersonalizedFeed(ctx, pulse, req)
	if err != nil {
		return nil, err
	}
	
	return feed.News, nil
}

// GetTrendingFeedNews возвращает трендовые новости для пульса
func (r *FeedRepository) GetTrendingFeedNews(ctx context.Context, pulse *models.UserPulse, limit int) ([]models.PersonalizedNews, error) {
	// Трендовые новости за последние 24 часа с высоким скором
	since := time.Now().Add(-24 * time.Hour)
	minScore := 0.7
	
	req := models.FeedRequest{
		PulseID:   pulse.ID,
		Page:      1,
		PageSize:  limit,
		DateFrom:  &since,
		MinScore:  &minScore,
		SortBy:    models.FeedSortByPersonalScore,
		SortOrder: models.SortOrderDesc,
	}
	
	feed, err := r.GetPersonalizedFeed(ctx, pulse, req)
	if err != nil {
		return nil, err
	}
	
	return feed.News, nil
}

// buildFeedQuery строит SQL запрос для получения персонализированной ленты
func (r *FeedRepository) buildFeedQuery(pulse *models.UserPulse, req models.FeedRequest) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	argIndex := 1
	
	baseQuery := `
		SELECT DISTINCT n.id, n.title, n.description, n.url, n.image_url, 
			   n.author, n.source_id, n.category_id, n.published_at,
			   n.relevance_score, n.view_count,
			   ns.name as source_name, ns.domain as source_domain, ns.logo_url as source_logo,
			   c.name as category_name, c.slug as category_slug, c.color as category_color
		FROM news n
		JOIN news_sources ns ON n.source_id = ns.id
		LEFT JOIN categories c ON n.category_id = c.id`
	
	// Базовое условие - только активные новости
	conditions = append(conditions, "n.is_active = true")
	
	// Условие для пульса - новости из источников ИЛИ категорий пульса
	var pulseConditions []string
	
	// Источники пульса
	if len(pulse.Sources) > 0 {
		sourceIDs := make([]string, len(pulse.Sources))
		for i, source := range pulse.Sources {
			sourceIDs[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, source.SourceID)
			argIndex++
		}
		pulseConditions = append(pulseConditions, fmt.Sprintf("n.source_id IN (%s)", strings.Join(sourceIDs, ",")))
	}
	
	// Категории пульса
	if len(pulse.Categories) > 0 {
		categoryIDs := make([]string, len(pulse.Categories))
		for i, category := range pulse.Categories {
			categoryIDs[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, category.CategoryID)
			argIndex++
		}
		pulseConditions = append(pulseConditions, fmt.Sprintf("n.category_id IN (%s)", strings.Join(categoryIDs, ",")))
	}
	
	if len(pulseConditions) > 0 {
		conditions = append(conditions, fmt.Sprintf("(%s)", strings.Join(pulseConditions, " OR ")))
	}
	
	// Фильтр по дате от
	if req.DateFrom != nil {
		conditions = append(conditions, fmt.Sprintf("n.published_at >= $%d", argIndex))
		args = append(args, *req.DateFrom)
		argIndex++
	}
	
	// Фильтр по дате до
	if req.DateTo != nil {
		conditions = append(conditions, fmt.Sprintf("n.published_at <= $%d", argIndex))
		args = append(args, *req.DateTo)
		argIndex++
	}
	
	// Фильтр по минимальному скору
	if req.MinScore != nil {
		conditions = append(conditions, fmt.Sprintf("n.relevance_score >= $%d", argIndex))
		args = append(args, *req.MinScore)
		argIndex++
	}
	
	// Добавляем условия WHERE
	if len(conditions) > 0 {
		baseQuery += " WHERE " + strings.Join(conditions, " AND ")
	}
	
	// Сортировка
	sortBy, sortOrder := models.NormalizeFeedSortParams(req.SortBy, req.SortOrder)
	
	// Для персонального скора используем формулу
	if sortBy == models.FeedSortByPersonalScore {
		baseQuery += fmt.Sprintf(" ORDER BY (n.relevance_score * CASE WHEN EXTRACT(EPOCH FROM (NOW() - n.published_at))/3600 <= 1 THEN 1.2 WHEN EXTRACT(EPOCH FROM (NOW() - n.published_at))/3600 <= 6 THEN 1.1 WHEN EXTRACT(EPOCH FROM (NOW() - n.published_at))/3600 <= 24 THEN 1.0 WHEN EXTRACT(EPOCH FROM (NOW() - n.published_at))/3600 <= 72 THEN 0.9 ELSE 0.8 END * CASE WHEN n.view_count > 1000 THEN 1.2 WHEN n.view_count > 500 THEN 1.1 WHEN n.view_count > 100 THEN 1.05 ELSE 1.0 END) %s", strings.ToUpper(sortOrder))
	} else {
		baseQuery += fmt.Sprintf(" ORDER BY n.%s %s", sortBy, strings.ToUpper(sortOrder))
	}
	
	// Лимит и оффсет
	baseQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, req.PageSize, req.GetOffset())
	
	return baseQuery, args
}

// buildFeedCountQuery строит запрос для подсчета общего количества новостей в ленте
func (r *FeedRepository) buildFeedCountQuery(pulse *models.UserPulse, req models.FeedRequest) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	argIndex := 1
	
	baseQuery := `
		SELECT COUNT(DISTINCT n.id)
		FROM news n
		JOIN news_sources ns ON n.source_id = ns.id`
	
	// Базовое условие - только активные новости
	conditions = append(conditions, "n.is_active = true")
	
	// Условие для пульса
	var pulseConditions []string
	
	if len(pulse.Sources) > 0 {
		sourceIDs := make([]string, len(pulse.Sources))
		for i, source := range pulse.Sources {
			sourceIDs[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, source.SourceID)
			argIndex++
		}
		pulseConditions = append(pulseConditions, fmt.Sprintf("n.source_id IN (%s)", strings.Join(sourceIDs, ",")))
	}
	
	if len(pulse.Categories) > 0 {
		categoryIDs := make([]string, len(pulse.Categories))
		for i, category := range pulse.Categories {
			categoryIDs[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, category.CategoryID)
			argIndex++
		}
		pulseConditions = append(pulseConditions, fmt.Sprintf("n.category_id IN (%s)", strings.Join(categoryIDs, ",")))
	}
	
	if len(pulseConditions) > 0 {
		conditions = append(conditions, fmt.Sprintf("(%s)", strings.Join(pulseConditions, " OR ")))
	}
	
	// Те же фильтры, что и в основном запросе
	if req.DateFrom != nil {
		conditions = append(conditions, fmt.Sprintf("n.published_at >= $%d", argIndex))
		args = append(args, *req.DateFrom)
		argIndex++
	}
	
	if req.DateTo != nil {
		conditions = append(conditions, fmt.Sprintf("n.published_at <= $%d", argIndex))
		args = append(args, *req.DateTo)
		argIndex++
	}
	
	if req.MinScore != nil {
		conditions = append(conditions, fmt.Sprintf("n.relevance_score >= $%d", argIndex))
		args = append(args, *req.MinScore)
		argIndex++
	}
	
	// Добавляем условия WHERE
	if len(conditions) > 0 {
		baseQuery += " WHERE " + strings.Join(conditions, " AND ")
	}
	
	return baseQuery, args
}

// determineMatchReason определяет причину попадания новости в ленту
func (r *FeedRepository) determineMatchReason(pulse *models.UserPulse, news models.PersonalizedNews) string {
	reasons := []string{}
	
	// Проверяем источники
	for _, source := range pulse.Sources {
		if source.SourceID == news.SourceID {
			reasons = append(reasons, fmt.Sprintf("источник: %s", news.SourceName))
			break
		}
	}
	
	// Проверяем категории
	if news.CategoryID != nil {
		for _, category := range pulse.Categories {
			if category.CategoryID == *news.CategoryID {
				reasons = append(reasons, fmt.Sprintf("категория: %s", news.CategoryName))
				break
			}
		}
	}
	
	if len(reasons) == 0 {
		return "общие критерии"
	}
	
	return strings.Join(reasons, ", ")
}

// calculateFeedStats вычисляет статистику персонализированной ленты
func (r *FeedRepository) calculateFeedStats(newsList []models.PersonalizedNews) models.PersonalizedFeedStats {
	if len(newsList) == 0 {
		return models.PersonalizedFeedStats{}
	}
	
	stats := models.PersonalizedFeedStats{
		TotalNews: len(newsList),
	}
	
	// Подсчет по источникам
	sourceMap := make(map[int]*models.SourceBreakdown)
	categoryMap := make(map[int]*models.CategoryBreakdown)
	totalScore := 0.0
	
	for _, news := range newsList {
		totalScore += news.PersonalScore
		
		// Источники
		if breakdown, exists := sourceMap[news.SourceID]; exists {
			breakdown.NewsCount++
		} else {
			sourceMap[news.SourceID] = &models.SourceBreakdown{
				SourceID:   news.SourceID,
				SourceName: news.SourceName,
				NewsCount:  1,
			}
		}
		
		// Категории
		if news.CategoryID != nil {
			if breakdown, exists := categoryMap[*news.CategoryID]; exists {
				breakdown.NewsCount++
			} else {
				categoryMap[*news.CategoryID] = &models.CategoryBreakdown{
					CategoryID:   *news.CategoryID,
					CategoryName: news.CategoryName,
					NewsCount:    1,
				}
			}
		}
	}
	
	// Преобразуем карты в слайсы и вычисляем проценты
	for _, breakdown := range sourceMap {
		breakdown.Percentage = float64(breakdown.NewsCount) / float64(stats.TotalNews) * 100
		stats.SourceBreakdown = append(stats.SourceBreakdown, *breakdown)
	}
	
	for _, breakdown := range categoryMap {
		breakdown.Percentage = float64(breakdown.NewsCount) / float64(stats.TotalNews) * 100
		stats.CategoryBreakdown = append(stats.CategoryBreakdown, *breakdown)
	}
	
	stats.NewsSources = len(sourceMap)
	stats.NewsCategories = len(categoryMap)
	stats.AverageScore = totalScore / float64(stats.TotalNews)
	
	return stats
}
