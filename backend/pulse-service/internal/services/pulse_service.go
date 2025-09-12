package services

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"pulse-service/internal/cache"
	"pulse-service/internal/config"
	"pulse-service/internal/database"
	"pulse-service/internal/models"
	"pulse-service/internal/repository"
)

// PulseService представляет сервис для работы с пульсами
type PulseService struct {
	pulseRepo *repository.PulseRepository
	feedRepo  *repository.FeedRepository
	cache     *cache.Cache
	config    *config.Config
	logger    *logrus.Logger
	db        *database.DB
}

// NewPulseService создает новый сервис пульсов
func NewPulseService(
	pulseRepo *repository.PulseRepository,
	feedRepo *repository.FeedRepository,
	cache *cache.Cache,
	config *config.Config,
	logger *logrus.Logger,
	db *database.DB,
) *PulseService {
	return &PulseService{
		pulseRepo: pulseRepo,
		feedRepo:  feedRepo,
		cache:     cache,
		config:    config,
		logger:    logger,
		db:        db,
	}
}

// GetDB возвращает подключение к базе данных
func (s *PulseService) GetDB() *database.DB {
	return s.db
}

// UpdateLastRefreshed обновляет время последнего обновления пульса
func (s *PulseService) UpdateLastRefreshed(ctx context.Context, pulseID string) error {
	query := `UPDATE user_pulses SET last_refreshed_at = NOW() WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, pulseID)
	if err != nil {
		return fmt.Errorf("failed to update last refreshed time: %w", err)
	}
	return nil
}

// CreatePulse создает новый пульс для пользователя
func (s *PulseService) CreatePulse(ctx context.Context, userID string, req models.PulseRequest) (*models.UserPulse, error) {
	// Валидируем запрос
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid pulse request: %w", err)
	}

	// Проверяем лимиты
	if err := s.checkUserLimits(ctx, userID); err != nil {
		return nil, err
	}

	// Создаем контекст с таймаутом
	timeoutCtx, cancel := context.WithTimeout(ctx, s.config.API.DBTimeout)
	defer cancel()

	// Создаем пульс
	pulse, err := s.pulseRepo.Create(timeoutCtx, userID, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create pulse: %w", err)
	}

	// Инвалидируем кеш пользователя
	s.cache.InvalidateUserCache(userID)

	s.logger.WithFields(logrus.Fields{
		"pulse_id": pulse.ID,
		"user_id":  userID,
		"name":     pulse.Name,
	}).Info("Pulse created")

	return pulse, nil
}

// GetPulseByID возвращает пульс по ID
func (s *PulseService) GetPulseByID(ctx context.Context, pulseID, userID string) (*models.UserPulse, error) {
	// Пытаемся получить из кеша
	cacheKey := cache.PulseCacheKey(pulseID, userID)
	var cachedPulse models.UserPulse

	if hit, err := s.cache.Get(cacheKey, &cachedPulse); err == nil && hit {
		s.logger.WithField("cache_key", cacheKey).Debug("Cache hit for pulse")
		return &cachedPulse, nil
	}

	// Создаем контекст с таймаутом
	timeoutCtx, cancel := context.WithTimeout(ctx, s.config.API.DBTimeout)
	defer cancel()

	// Получаем из базы данных
	pulse, err := s.pulseRepo.GetByID(timeoutCtx, pulseID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get pulse: %w", err)
	}

	// Кешируем результат
	if err := s.cache.Set(cacheKey, pulse, s.config.Caching.UserPulsesTTL); err != nil {
		s.logger.WithError(err).Warn("Failed to cache pulse")
	}

	return pulse, nil
}

// GetUserPulses возвращает все пульсы пользователя
func (s *PulseService) GetUserPulses(ctx context.Context, userID string, filter models.PulseFilter) ([]models.UserPulse, error) {
	// Валидируем фильтр
	if err := filter.Validate(s.config.API.MaxPulsesPerUser); err != nil {
		return nil, fmt.Errorf("invalid filter: %w", err)
	}

	// Устанавливаем значения по умолчанию
	if filter.PageSize == 0 {
		filter.PageSize = 20
	}
	if filter.Page == 0 {
		filter.Page = 1
	}

	// Пытаемся получить из кеша
	filterHash := s.buildFilterHash(filter)
	cacheKey := cache.UserPulsesCacheKey(userID, filterHash)
	var cachedPulses []models.UserPulse

	if hit, err := s.cache.Get(cacheKey, &cachedPulses); err == nil && hit {
		s.logger.WithField("cache_key", cacheKey).Debug("Cache hit for user pulses")
		return cachedPulses, nil
	}

	// Создаем контекст с таймаутом
	timeoutCtx, cancel := context.WithTimeout(ctx, s.config.API.DBTimeout)
	defer cancel()

	// Получаем из базы данных
	pulses, err := s.pulseRepo.GetByUserID(timeoutCtx, userID, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get user pulses: %w", err)
	}

	// Кешируем результат
	if err := s.cache.Set(cacheKey, pulses, s.config.Caching.UserPulsesTTL); err != nil {
		s.logger.WithError(err).Warn("Failed to cache user pulses")
	}

	return pulses, nil
}

// GetDefaultPulse возвращает дефолтный пульс пользователя
func (s *PulseService) GetDefaultPulse(ctx context.Context, userID string) (*models.UserPulse, error) {
	// Пытаемся получить из кеша
	cacheKey := cache.DefaultPulseCacheKey(userID)
	var cachedPulse models.UserPulse

	if hit, err := s.cache.Get(cacheKey, &cachedPulse); err == nil && hit {
		s.logger.WithField("cache_key", cacheKey).Debug("Cache hit for default pulse")
		return &cachedPulse, nil
	}

	// Создаем контекст с таймаутом
	timeoutCtx, cancel := context.WithTimeout(ctx, s.config.API.DBTimeout)
	defer cancel()

	// Получаем из базы данных
	pulse, err := s.pulseRepo.GetDefaultByUserID(timeoutCtx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get default pulse: %w", err)
	}

	// Кешируем результат
	if err := s.cache.Set(cacheKey, pulse, s.config.Caching.UserPulsesTTL); err != nil {
		s.logger.WithError(err).Warn("Failed to cache default pulse")
	}

	return pulse, nil
}

// UpdatePulse обновляет пульс пользователя
func (s *PulseService) UpdatePulse(ctx context.Context, pulseID, userID string, req models.PulseRequest) (*models.UserPulse, error) {
	// Валидируем запрос
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid pulse request: %w", err)
	}

	// Создаем контекст с таймаутом
	timeoutCtx, cancel := context.WithTimeout(ctx, s.config.API.DBTimeout)
	defer cancel()

	// Обновляем пульс
	pulse, err := s.pulseRepo.Update(timeoutCtx, pulseID, userID, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update pulse: %w", err)
	}

	// Инвалидируем кеш
	s.cache.InvalidateUserCache(userID)
	s.cache.InvalidatePulseCache(pulseID)

	s.logger.WithFields(logrus.Fields{
		"pulse_id": pulseID,
		"user_id":  userID,
		"name":     pulse.Name,
	}).Info("Pulse updated")

	return pulse, nil
}

// DeletePulse удаляет пульс пользователя
func (s *PulseService) DeletePulse(ctx context.Context, pulseID, userID string) error {
	// Создаем контекст с таймаутом
	timeoutCtx, cancel := context.WithTimeout(ctx, s.config.API.DBTimeout)
	defer cancel()

	// Удаляем пульс
	if err := s.pulseRepo.Delete(timeoutCtx, pulseID, userID); err != nil {
		return fmt.Errorf("failed to delete pulse: %w", err)
	}

	// Инвалидируем кеш
	s.cache.InvalidateUserCache(userID)
	s.cache.InvalidatePulseCache(pulseID)

	s.logger.WithFields(logrus.Fields{
		"pulse_id": pulseID,
		"user_id":  userID,
	}).Info("Pulse deleted")

	return nil
}

// GetPersonalizedFeed возвращает персонализированную ленту новостей
func (s *PulseService) GetPersonalizedFeed(ctx context.Context, pulseID, userID string, req models.FeedRequest) (*models.PersonalizedFeed, error) {
	// Валидируем запрос
	if err := req.Validate(s.config.API.MaxNewsPerFeed, s.config.API.MaxNewsPerFeed); err != nil {
		return nil, fmt.Errorf("invalid feed request: %w", err)
	}

	// Устанавливаем значения по умолчанию
	if req.PageSize == 0 {
		req.PageSize = s.config.API.DefaultFeedPageSize
	}
	if req.Page == 0 {
		req.Page = 1
	}

	// Получаем пульс
	pulse, err := s.GetPulseByID(ctx, pulseID, userID)
	if err != nil {
		return nil, err
	}

	// Пытаемся получить из кеша
	requestHash := s.buildFeedRequestHash(req)
	cacheKey := cache.PersonalizedFeedCacheKey(pulseID, requestHash)
	var cachedFeed models.PersonalizedFeed

	if hit, err := s.cache.Get(cacheKey, &cachedFeed); err == nil && hit {
		s.logger.WithField("cache_key", cacheKey).Debug("Cache hit for personalized feed")
		return &cachedFeed, nil
	}

	// Создаем контекст с таймаутом
	timeoutCtx, cancel := context.WithTimeout(ctx, s.config.API.DBTimeout)
	defer cancel()

	// Получаем из базы данных
	feed, err := s.feedRepo.GetPersonalizedFeed(timeoutCtx, pulse, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get personalized feed: %w", err)
	}

	// Обновляем время последнего обновления пульса
	if pulse.NeedsRefresh() {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := s.pulseRepo.UpdateLastRefreshed(ctx, pulseID); err != nil {
				s.logger.WithError(err).WithField("pulse_id", pulseID).Warn("Failed to update last refreshed time")
			}
		}()
	}

	// Кешируем результат
	if err := s.cache.Set(cacheKey, feed, s.config.Caching.PersonalizedFeedTTL); err != nil {
		s.logger.WithError(err).Warn("Failed to cache personalized feed")
	}

	return feed, nil
}

// GetLatestFeedNews возвращает последние новости для пульса
func (s *PulseService) GetLatestFeedNews(ctx context.Context, pulseID, userID string, limit int) ([]models.PersonalizedNews, error) {
	if limit <= 0 || limit > s.config.API.MaxNewsPerFeed {
		limit = s.config.API.DefaultFeedPageSize
	}

	// Получаем пульс
	pulse, err := s.GetPulseByID(ctx, pulseID, userID)
	if err != nil {
		return nil, err
	}

	// Пытаемся получить из кеша
	cacheKey := cache.LatestFeedCacheKey(pulseID, limit)
	var cachedNews []models.PersonalizedNews

	if hit, err := s.cache.Get(cacheKey, &cachedNews); err == nil && hit {
		s.logger.WithField("cache_key", cacheKey).Debug("Cache hit for latest feed news")
		return cachedNews, nil
	}

	// Создаем контекст с таймаутом
	timeoutCtx, cancel := context.WithTimeout(ctx, s.config.API.DBTimeout)
	defer cancel()

	// Получаем из базы данных
	news, err := s.feedRepo.GetLatestFeedNews(timeoutCtx, pulse, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest feed news: %w", err)
	}

	// Кешируем результат
	if err := s.cache.Set(cacheKey, news, s.config.Caching.PersonalizedFeedTTL); err != nil {
		s.logger.WithError(err).Warn("Failed to cache latest feed news")
	}

	return news, nil
}

// GetTrendingFeedNews возвращает трендовые новости для пульса
func (s *PulseService) GetTrendingFeedNews(ctx context.Context, pulseID, userID string, limit int) ([]models.PersonalizedNews, error) {
	if limit <= 0 || limit > s.config.API.MaxNewsPerFeed {
		limit = s.config.API.DefaultFeedPageSize
	}

	// Получаем пульс
	pulse, err := s.GetPulseByID(ctx, pulseID, userID)
	if err != nil {
		return nil, err
	}

	// Пытаемся получить из кеша
	cacheKey := cache.TrendingFeedCacheKey(pulseID, limit)
	var cachedNews []models.PersonalizedNews

	if hit, err := s.cache.Get(cacheKey, &cachedNews); err == nil && hit {
		s.logger.WithField("cache_key", cacheKey).Debug("Cache hit for trending feed news")
		return cachedNews, nil
	}

	// Создаем контекст с таймаутом
	timeoutCtx, cancel := context.WithTimeout(ctx, s.config.API.DBTimeout)
	defer cancel()

	// Получаем из базы данных
	news, err := s.feedRepo.GetTrendingFeedNews(timeoutCtx, pulse, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get trending feed news: %w", err)
	}

	// Кешируем результат
	if err := s.cache.Set(cacheKey, news, s.config.Caching.PersonalizedFeedTTL); err != nil {
		s.logger.WithError(err).Warn("Failed to cache trending feed news")
	}

	return news, nil
}

// checkUserLimits проверяет лимиты пользователя
func (s *PulseService) checkUserLimits(ctx context.Context, userID string) error {
	// Подсчитываем количество активных пульсов пользователя
	filter := models.PulseFilter{
		UserID:   &userID,
		IsActive: &[]bool{true}[0],
		Page:     1,
		PageSize: s.config.API.MaxPulsesPerUser + 1, // +1 для проверки лимита
	}

	pulses, err := s.pulseRepo.GetByUserID(ctx, userID, filter)
	if err != nil {
		return fmt.Errorf("failed to check user pulse count: %w", err)
	}

	if len(pulses) >= s.config.API.MaxPulsesPerUser {
		return fmt.Errorf("maximum number of pulses (%d) reached for user", s.config.API.MaxPulsesPerUser)
	}

	return nil
}

// buildFilterHash создает хеш для фильтра пульсов
func (s *PulseService) buildFilterHash(filter models.PulseFilter) string {
	// Сериализуем фильтр в JSON для создания уникального ключа
	data, err := json.Marshal(filter)
	if err != nil {
		// Fallback - создаем простой ключ
		return fmt.Sprintf("%d_%d", filter.Page, filter.PageSize)
	}

	// Создаем MD5 хеш для компактности
	hash := md5.Sum(data)
	return fmt.Sprintf("%x", hash)
}

// buildFeedRequestHash создает хеш для запроса персонализированной ленты
func (s *PulseService) buildFeedRequestHash(req models.FeedRequest) string {
	// Сериализуем запрос в JSON для создания уникального ключа
	data, err := json.Marshal(req)
	if err != nil {
		// Fallback - создаем простой ключ
		return fmt.Sprintf("%d_%d_%d", req.PulseID, req.Page, req.PageSize)
	}

	// Создаем MD5 хеш для компактности
	hash := md5.Sum(data)
	return fmt.Sprintf("%x", hash)
}

// ClearCache очищает кеш
func (s *PulseService) ClearCache() {
	s.cache.Clear()
	s.logger.Info("Pulse service cache cleared")
}

// GetCacheStats возвращает статистику кеша
func (s *PulseService) GetCacheStats() map[string]interface{} {
	return s.cache.GetStats()
}

// CollectPulseNews собирает новости для пульса и добавляет их в таблицу pulse_news
func (s *PulseService) CollectPulseNews(ctx context.Context, pulseID string) error {
	// ОТЛАДОЧНЫЙ ЛОГ В САМОМ НАЧАЛЕ
	s.logger.Info("=== CollectPulseNews FUNCTION START ===")
	s.logger.WithField("pulse_id", pulseID).Info("=== CollectPulseNews CALLED ===")

	// Проверяем, что пульс существует
	exists, err := s.pulseRepo.PulseExists(ctx, pulseID)
	if err != nil {
		return fmt.Errorf("failed to check pulse existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("pulse with id %s not found", pulseID)
	}

	// Получаем пульс для извлечения ключевых слов
	pulse, err := s.pulseRepo.GetByIDWithoutUser(ctx, pulseID)
	if err != nil {
		return fmt.Errorf("failed to get pulse details: %w", err)
	}

	// Получаем источники и категории пульса
	sources, err := s.pulseRepo.GetPulseSources(ctx, pulseID)
	if err != nil {
		return fmt.Errorf("failed to get pulse sources: %w", err)
	}

	categories, err := s.pulseRepo.GetPulseCategories(ctx, pulseID)
	if err != nil {
		return fmt.Errorf("failed to get pulse categories: %w", err)
	}

	// Собираем ID источников и категорий
	sourceIDs := make([]int, len(sources))
	for i, source := range sources {
		sourceIDs[i] = source.SourceID
	}

	categoryIDs := make([]int, len(categories))
	for i, category := range categories {
		categoryIDs[i] = category.CategoryID
	}

	// Извлекаем ключевые слова из поля keywords пульса
	var keywords []string
	if pulse.Keywords != "" {
		// Если есть ключевые слова, используем их
		keywords = extractKeywords(pulse.Keywords)
	} else {
		// Иначе извлекаем из названия и описания
		keywords = extractKeywords(pulse.Name + " " + pulse.Description)
	}

	// Отладочная информация
	s.logger.WithFields(logrus.Fields{
		"pulse_id":           pulseID,
		"pulse_name":         pulse.Name,
		"pulse_keywords":     pulse.Keywords,
		"extracted_keywords": keywords,
		"source_count":       len(sourceIDs),
		"category_count":     len(categoryIDs),
		"source_ids":         sourceIDs,
		"category_ids":       categoryIDs,
	}).Info("DEBUG: CollectPulseNews parameters")

	// Если ключевые слова все еще пустые, не добавляем новости
	if len(keywords) == 0 {
		s.logger.WithField("pulse_id", pulseID).Warn("No keywords found for pulse, skipping news collection")
		return nil
	}

	// Если нет категорий, не добавляем новости
	if len(categoryIDs) == 0 {
		s.logger.WithFields(logrus.Fields{
			"pulse_id":       pulseID,
			"category_count": len(categoryIDs),
		}).Warn("Missing categories for pulse, skipping news collection")
		return nil
	}

	// Создаем динамический SQL запрос

	// Создаем плейсхолдеры для категорий (начинаем с $2, так как $1 - это pulse_id)
	categoryPlaceholders := make([]string, len(categoryIDs))
	for i := range categoryIDs {
		categoryPlaceholders[i] = fmt.Sprintf("$%d", i+2)
	}

	// Создаем плейсхолдеры для ключевых слов в контенте, заголовке и описании
	keywordConditions := make([]string, len(keywords))
	for i := range keywords {
		contentPlaceholder := fmt.Sprintf("$%d", i*3+2+len(categoryIDs))
		titlePlaceholder := fmt.Sprintf("$%d", i*3+3+len(categoryIDs))
		descPlaceholder := fmt.Sprintf("$%d", i*3+4+len(categoryIDs))
		keywordConditions[i] = fmt.Sprintf("(LOWER(n.content) LIKE %s OR LOWER(n.title) LIKE %s OR LOWER(n.description) LIKE %s)", contentPlaceholder, titlePlaceholder, descPlaceholder)
	}

	// Строим SQL запрос с учетом категорий пульса
	var categoryCondition string
	if len(categoryIDs) > 0 {
		// Создаем плейсхолдеры для категорий
		categoryPlaceholders := make([]string, len(categoryIDs))
		for i := range categoryIDs {
			categoryPlaceholders[i] = fmt.Sprintf("$%d", i+2) // +2 потому что $1 это pulse_id
		}
		categoryCondition = fmt.Sprintf("AND n.category_id IN (%s)", strings.Join(categoryPlaceholders, ","))
	} else {
		categoryCondition = "" // Если нет категорий, берем все новости
	}

	// Строим условие для ключевых слов
	var keywordCondition string
	if len(keywordConditions) > 0 {
		keywordCondition = fmt.Sprintf("AND (%s)", strings.Join(keywordConditions, " OR "))
	}

	query := fmt.Sprintf(`
		INSERT INTO pulse_news (pulse_id, news_id, relevance_score, match_reason)
		SELECT
			$1::uuid as pulse_id,
			n.id as news_id,
			1.0 as relevance_score,
			'keyword_match' as match_reason
		FROM news n
		WHERE n.is_active = true
		AND n.published_at >= NOW() - INTERVAL '30 days'
		%s
		%s
		AND NOT EXISTS (
			SELECT 1 FROM pulse_news pn
			WHERE pn.pulse_id = $1::uuid
			AND pn.news_id = n.id
		)
		ORDER BY n.published_at DESC
		LIMIT 100
		ON CONFLICT (pulse_id, news_id) DO NOTHING
	`, categoryCondition, keywordCondition)

	// Подготавливаем параметры для запроса
	args := []interface{}{pulseID}
	// Добавляем категории в параметры
	for _, categoryID := range categoryIDs {
		args = append(args, categoryID)
	}
	// Добавляем ключевые слова в параметры (для контента, заголовка и описания)
	for _, keyword := range keywords {
		keywordPattern := "%" + strings.ToLower(keyword) + "%"
		args = append(args, keywordPattern, keywordPattern, keywordPattern)
	}

	// КРИТИЧЕСКАЯ ОТЛАДОЧНАЯ ИНФОРМАЦИЯ
	s.logger.Error("=== CRITICAL: SQL EXECUTION STARTING ===")
	s.logger.WithFields(logrus.Fields{
		"generated_sql": query,
		"args_count":    len(args),
		"args":          args,
	}).Error("=== DEBUG: Generated SQL query ===")

	// Выполняем запрос
	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		s.logger.WithError(err).Error("=== CRITICAL: SQL EXECUTION FAILED ===")
		return fmt.Errorf("failed to collect pulse news: %w", err)
	}

	// Проверяем результат выполнения
	rowsAffected, _ := result.RowsAffected()
	s.logger.WithField("rows_affected", rowsAffected).Error("=== CRITICAL: SQL EXECUTION COMPLETED ===")

	s.logger.WithFields(logrus.Fields{
		"pulse_id":       pulseID,
		"source_count":   len(sourceIDs),
		"category_count": len(categoryIDs),
		"keyword_count":  len(keywords),
		"source_ids":     sourceIDs,
		"category_ids":   categoryIDs,
		"keywords":       keywords,
	}).Info("Pulse news collected")

	// ДОПОЛНИТЕЛЬНЫЙ ОТЛАДОЧНЫЙ ЛОГ
	s.logger.WithFields(logrus.Fields{
		"pulse_id":     pulseID,
		"source_ids":   sourceIDs,
		"category_ids": categoryIDs,
		"keywords":     keywords,
	}).Info("=== DEBUG: CollectPulseNews parameters ===")

	return nil
}

// extractKeywords извлекает ключевые слова из текста
func extractKeywords(text string) []string {
	// Удаляем знаки препинания и приводим к нижнему регистру
	reg := regexp.MustCompile(`[^\p{L}\p{N}\s]`)
	cleanText := reg.ReplaceAllString(strings.ToLower(text), " ")

	// Разбиваем на слова
	words := strings.Fields(cleanText)

	// Фильтруем стоп-слова и короткие слова
	stopWords := map[string]bool{
		"и": true, "в": true, "на": true, "с": true, "по": true, "для": true,
		"от": true, "до": true, "из": true, "к": true, "о": true, "об": true,
		"что": true, "как": true, "где": true, "когда": true, "почему": true,
		"это": true, "этот": true, "эта": true, "эти": true,
		"новости": true, "новость": true, "новый": true, "новые": true,
		"пульс": true, "пульса": true, "пульсы": true,
	}

	var keywords []string
	for _, word := range words {
		if len(word) >= 3 && !stopWords[word] {
			keywords = append(keywords, "%"+word+"%")
		}
	}

	return keywords
}
