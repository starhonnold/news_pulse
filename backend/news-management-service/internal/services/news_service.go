package services

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"news-management-service/internal/cache"
	"news-management-service/internal/config"
	"news-management-service/internal/models"
	"news-management-service/internal/repository"
)

// NewsService представляет сервис для работы с новостями
type NewsService struct {
	newsRepo     *repository.NewsRepository
	categoryRepo *repository.CategoryRepository
	sourceRepo   *repository.SourceRepository
	countryRepo  *repository.CountryRepository
	cache        *cache.Cache
	config       *config.Config
	logger       *logrus.Logger
}

// NewNewsService создает новый сервис новостей
func NewNewsService(
	newsRepo *repository.NewsRepository,
	categoryRepo *repository.CategoryRepository,
	sourceRepo *repository.SourceRepository,
	countryRepo *repository.CountryRepository,
	cache *cache.Cache,
	config *config.Config,
	logger *logrus.Logger,
) *NewsService {
	return &NewsService{
		newsRepo:     newsRepo,
		categoryRepo: categoryRepo,
		sourceRepo:   sourceRepo,
		countryRepo:  countryRepo,
		cache:        cache,
		config:       config,
		logger:       logger,
	}
}

// GetNewsByID возвращает новость по ID
func (s *NewsService) GetNewsByID(ctx context.Context, id int) (*models.News, error) {
	// Создаем контекст с таймаутом
	timeoutCtx, cancel := context.WithTimeout(ctx, s.config.API.DBTimeout)
	defer cancel()
	
	news, err := s.newsRepo.GetByID(timeoutCtx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get news by id: %w", err)
	}
	
	// Увеличиваем счетчик просмотров асинхронно
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		if err := s.newsRepo.UpdateViewCount(ctx, id); err != nil {
			s.logger.WithError(err).WithField("news_id", id).Warn("Failed to update view count")
		}
	}()
	
	return news, nil
}

// GetNewsByFilter возвращает новости по фильтру
func (s *NewsService) GetNewsByFilter(ctx context.Context, filter models.NewsFilter) (*models.NewsResponse, error) {
	// Валидируем фильтр
	if err := filter.Validate(s.config.API.MaxPageSize); err != nil {
		return nil, fmt.Errorf("invalid filter: %w", err)
	}
	
	// Устанавливаем значения по умолчанию
	if filter.PageSize == 0 {
		filter.PageSize = s.config.API.DefaultPageSize
	}
	if filter.Page == 0 {
		filter.Page = 1
	}
	
	// Нормализуем параметры сортировки
	filter.SortBy, filter.SortOrder = models.NormalizeSortParams(filter.SortBy, filter.SortOrder)
	
	// Пытаемся получить из кеша
	cacheKey := s.buildFilterCacheKey(filter)
	var cachedResponse models.NewsResponse
	
	if hit, err := s.cache.Get(cacheKey, &cachedResponse); err == nil && hit {
		s.logger.WithField("cache_key", cacheKey).Debug("Cache hit for news filter")
		return &cachedResponse, nil
	}
	
	// Создаем контекст с таймаутом
	timeoutCtx, cancel := context.WithTimeout(ctx, s.config.API.DBTimeout)
	defer cancel()
	
	// Получаем из базы данных
	response, err := s.newsRepo.GetByFilter(timeoutCtx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get news by filter: %w", err)
	}
	
	// Кешируем результат
	if err := s.cache.Set(cacheKey, response, s.config.Caching.NewsTTL); err != nil {
		s.logger.WithError(err).Warn("Failed to cache news response")
	}
	
	return response, nil
}

// GetLatestNews возвращает последние новости
func (s *NewsService) GetLatestNews(ctx context.Context, limit int) ([]models.News, error) {
	if limit <= 0 || limit > s.config.API.MaxPageSize {
		limit = s.config.API.DefaultPageSize
	}
	
	// Пытаемся получить из кеша
	cacheKey := fmt.Sprintf("latest_news:%d", limit)
	var cachedNews []models.News
	
	if hit, err := s.cache.Get(cacheKey, &cachedNews); err == nil && hit {
		s.logger.WithField("cache_key", cacheKey).Debug("Cache hit for latest news")
		return cachedNews, nil
	}
	
	// Создаем контекст с таймаутом
	timeoutCtx, cancel := context.WithTimeout(ctx, s.config.API.DBTimeout)
	defer cancel()
	
	// Получаем из базы данных
	news, err := s.newsRepo.GetLatest(timeoutCtx, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest news: %w", err)
	}
	
	// Кешируем результат
	if err := s.cache.Set(cacheKey, news, s.config.Caching.NewsTTL); err != nil {
		s.logger.WithError(err).Warn("Failed to cache latest news")
	}
	
	return news, nil
}

// GetTrendingNews возвращает трендовые новости
func (s *NewsService) GetTrendingNews(ctx context.Context, limit int) ([]models.News, error) {
	if limit <= 0 || limit > s.config.API.MaxPageSize {
		limit = s.config.API.DefaultPageSize
	}
	
	// Пытаемся получить из кеша
	cacheKey := fmt.Sprintf("trending_news:%d", limit)
	var cachedNews []models.News
	
	if hit, err := s.cache.Get(cacheKey, &cachedNews); err == nil && hit {
		s.logger.WithField("cache_key", cacheKey).Debug("Cache hit for trending news")
		return cachedNews, nil
	}
	
	// Создаем контекст с таймаутом
	timeoutCtx, cancel := context.WithTimeout(ctx, s.config.API.DBTimeout)
	defer cancel()
	
	// Получаем из базы данных
	news, err := s.newsRepo.GetTrending(timeoutCtx, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get trending news: %w", err)
	}
	
	// Кешируем результат
	if err := s.cache.Set(cacheKey, news, s.config.Caching.NewsTTL); err != nil {
		s.logger.WithError(err).Warn("Failed to cache trending news")
	}
	
	return news, nil
}

// SearchNews выполняет поиск новостей
func (s *NewsService) SearchNews(ctx context.Context, query string, page, pageSize int) (*models.SearchResult, error) {
	// Валидируем параметры
	if len(query) > s.config.API.MaxSearchLength {
		return nil, fmt.Errorf("search query too long")
	}
	
	if pageSize <= 0 || pageSize > s.config.API.MaxPageSize {
		pageSize = s.config.API.DefaultPageSize
	}
	
	if page <= 0 {
		page = 1
	}
	
	// Пытаемся получить из кеша
	cacheKey := cache.SearchCacheKey(query, page, pageSize)
	var cachedResult models.SearchResult
	
	if hit, err := s.cache.Get(cacheKey, &cachedResult); err == nil && hit {
		s.logger.WithField("cache_key", cacheKey).Debug("Cache hit for search")
		return &cachedResult, nil
	}
	
	// Создаем контекст с таймаутом
	timeoutCtx, cancel := context.WithTimeout(ctx, s.config.API.DBTimeout)
	defer cancel()
	
	// Выполняем поиск
	result, err := s.newsRepo.Search(timeoutCtx, query, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to search news: %w", err)
	}
	
	// Кешируем результат
	if err := s.cache.Set(cacheKey, result, s.config.Caching.NewsTTL); err != nil {
		s.logger.WithError(err).Warn("Failed to cache search result")
	}
	
	return result, nil
}

// GetCategories возвращает все категории
func (s *NewsService) GetCategories(ctx context.Context) ([]models.Category, error) {
	// Пытаемся получить из кеша
	cacheKey := cache.CategoriesCacheKey()
	var cachedCategories []models.Category
	
	if hit, err := s.cache.Get(cacheKey, &cachedCategories); err == nil && hit {
		s.logger.WithField("cache_key", cacheKey).Debug("Cache hit for categories")
		return cachedCategories, nil
	}
	
	// Создаем контекст с таймаутом
	timeoutCtx, cancel := context.WithTimeout(ctx, s.config.API.DBTimeout)
	defer cancel()
	
	// Получаем из базы данных
	categories, err := s.categoryRepo.GetAll(timeoutCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}
	
	// Кешируем результат
	if err := s.cache.Set(cacheKey, categories, s.config.Caching.CategoriesTTL); err != nil {
		s.logger.WithError(err).Warn("Failed to cache categories")
	}
	
	return categories, nil
}

// GetSources возвращает все источники новостей
func (s *NewsService) GetSources(ctx context.Context) ([]models.NewsSource, error) {
	// Пытаемся получить из кеша
	cacheKey := cache.SourcesCacheKey()
	var cachedSources []models.NewsSource
	
	if hit, err := s.cache.Get(cacheKey, &cachedSources); err == nil && hit {
		s.logger.WithField("cache_key", cacheKey).Debug("Cache hit for sources")
		return cachedSources, nil
	}
	
	// Создаем контекст с таймаутом
	timeoutCtx, cancel := context.WithTimeout(ctx, s.config.API.DBTimeout)
	defer cancel()
	
	// Получаем из базы данных
	sources, err := s.sourceRepo.GetAll(timeoutCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to get sources: %w", err)
	}
	
	// Кешируем результат
	if err := s.cache.Set(cacheKey, sources, s.config.Caching.SourcesTTL); err != nil {
		s.logger.WithError(err).Warn("Failed to cache sources")
	}
	
	return sources, nil
}

// GetCountries возвращает все страны
func (s *NewsService) GetCountries(ctx context.Context) ([]models.Country, error) {
	// Пытаемся получить из кеша
	cacheKey := cache.CountriesCacheKey()
	var cachedCountries []models.Country
	
	if hit, err := s.cache.Get(cacheKey, &cachedCountries); err == nil && hit {
		s.logger.WithField("cache_key", cacheKey).Debug("Cache hit for countries")
		return cachedCountries, nil
	}
	
	// Создаем контекст с таймаутом
	timeoutCtx, cancel := context.WithTimeout(ctx, s.config.API.DBTimeout)
	defer cancel()
	
	// Получаем из базы данных
	countries, err := s.countryRepo.GetAll(timeoutCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to get countries: %w", err)
	}
	
	// Кешируем результат
	if err := s.cache.Set(cacheKey, countries, s.config.Caching.SourcesTTL); err != nil {
		s.logger.WithError(err).Warn("Failed to cache countries")
	}
	
	return countries, nil
}

// GetStats возвращает статистику новостей
func (s *NewsService) GetStats(ctx context.Context) (*models.NewsStats, error) {
	// Пытаемся получить из кеша
	cacheKey := cache.StatsCacheKey()
	var cachedStats models.NewsStats
	
	if hit, err := s.cache.Get(cacheKey, &cachedStats); err == nil && hit {
		s.logger.WithField("cache_key", cacheKey).Debug("Cache hit for stats")
		return &cachedStats, nil
	}
	
	// Создаем контекст с таймаутом
	timeoutCtx, cancel := context.WithTimeout(ctx, s.config.API.DBTimeout)
	defer cancel()
	
	// Собираем статистику из разных источников
	stats := &models.NewsStats{}
	
	// Получаем топ категорий
	topCategories, err := s.categoryRepo.GetTopCategories(timeoutCtx, 10)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to get top categories")
	} else {
		stats.TopCategories = topCategories
	}
	
	// Получаем топ источников
	topSources, err := s.sourceRepo.GetTopSources(timeoutCtx, 10)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to get top sources")
	} else {
		stats.TopSources = topSources
	}
	
	// Получаем топ стран
	topCountries, err := s.countryRepo.GetTopCountries(timeoutCtx, 10)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to get top countries")
	} else {
		stats.TopCountries = topCountries
	}
	
	// Получаем последние новости
	recentNews, err := s.newsRepo.GetLatest(timeoutCtx, 10)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to get recent news")
	} else {
		stats.RecentNews = recentNews
	}
	
	// Получаем трендовые новости
	trendingNews, err := s.newsRepo.GetTrending(timeoutCtx, 10)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to get trending news")
	} else {
		stats.TrendingNews = trendingNews
	}
	
	// Кешируем результат
	if err := s.cache.Set(cacheKey, stats, s.config.Caching.NewsTTL); err != nil {
		s.logger.WithError(err).Warn("Failed to cache stats")
	}
	
	return stats, nil
}

// buildFilterCacheKey создает ключ кеша для фильтра новостей
func (s *NewsService) buildFilterCacheKey(filter models.NewsFilter) string {
	// Сериализуем фильтр в JSON для создания уникального ключа
	data, err := json.Marshal(filter)
	if err != nil {
		// Fallback - создаем простой ключ
		return fmt.Sprintf("news_filter:%d:%d", filter.Page, filter.PageSize)
	}
	
	// Создаем MD5 хеш для компактности
	hash := md5.Sum(data)
	return fmt.Sprintf("news_filter:%x", hash)
}

// ClearCache очищает кеш
func (s *NewsService) ClearCache() {
	s.cache.Clear()
	s.logger.Info("News service cache cleared")
}

// GetCacheStats возвращает статистику кеша
func (s *NewsService) GetCacheStats() map[string]interface{} {
	return s.cache.GetStats()
}
