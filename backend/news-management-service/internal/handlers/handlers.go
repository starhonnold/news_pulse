package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"news-management-service/internal/models"
	"news-management-service/internal/services"
)

// Handler представляет HTTP обработчики
type Handler struct {
	newsService *services.NewsService
	logger      *logrus.Logger
}

// NewHandler создает новый обработчик
func NewHandler(newsService *services.NewsService, logger *logrus.Logger) *Handler {
	return &Handler{
		newsService: newsService,
		logger:      logger,
	}
}

// Response представляет стандартный ответ API
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

// ErrorResponse представляет ответ с ошибкой
type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
}

// HealthCheck возвращает статус здоровья сервиса
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"service":   "news-management-service",
		"version":   "1.0.0",
		"uptime":    time.Since(startTime),
	}

	// Добавляем статистику кеша
	health["cache"] = h.newsService.GetCacheStats()

	h.sendResponse(w, http.StatusOK, Response{
		Success: true,
		Data:    health,
	})
}

// GetNews возвращает список новостей с фильтрацией и пагинацией
func (h *Handler) GetNews(w http.ResponseWriter, r *http.Request) {
	// Парсим параметры фильтра
	filter, err := h.parseNewsFilter(r)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid filter parameters", err.Error())
		return
	}

	// Получаем новости
	response, err := h.newsService.GetNewsByFilter(r.Context(), filter)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get news")
		h.sendError(w, http.StatusInternalServerError, "Failed to get news", "")
		return
	}

	h.sendResponse(w, http.StatusOK, Response{
		Success: true,
		Data:    response,
	})
}

// GetNewsById возвращает новость по ID
func (h *Handler) GetNewsById(w http.ResponseWriter, r *http.Request) {
	// Извлекаем ID из URL
	idStr := h.extractIDFromPath(r.URL.Path, "/api/news/")
	if idStr == "" {
		h.sendError(w, http.StatusBadRequest, "News ID is required", "")
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid news ID", err.Error())
		return
	}

	// Получаем новость
	news, err := h.newsService.GetNewsByID(r.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.sendError(w, http.StatusNotFound, "News not found", "")
		} else {
			h.logger.WithError(err).WithField("news_id", id).Error("Failed to get news")
			h.sendError(w, http.StatusInternalServerError, "Failed to get news", "")
		}
		return
	}

	h.sendResponse(w, http.StatusOK, Response{
		Success: true,
		Data:    news,
	})
}

// GetLatestNews возвращает последние новости
func (h *Handler) GetLatestNews(w http.ResponseWriter, r *http.Request) {
	// Парсим лимит
	limit := 20
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	// Получаем последние новости
	news, err := h.newsService.GetLatestNews(r.Context(), limit)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get latest news")
		h.sendError(w, http.StatusInternalServerError, "Failed to get latest news", "")
		return
	}

	h.sendResponse(w, http.StatusOK, Response{
		Success: true,
		Data:    news,
	})
}

// GetTrendingNews возвращает трендовые новости
func (h *Handler) GetTrendingNews(w http.ResponseWriter, r *http.Request) {
	// Парсим лимит
	limit := 20
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	// Получаем трендовые новости
	news, err := h.newsService.GetTrendingNews(r.Context(), limit)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get trending news")
		h.sendError(w, http.StatusInternalServerError, "Failed to get trending news", "")
		return
	}

	h.sendResponse(w, http.StatusOK, Response{
		Success: true,
		Data:    news,
	})
}

// SearchNews выполняет поиск новостей
func (h *Handler) SearchNews(w http.ResponseWriter, r *http.Request) {
	// Получаем поисковый запрос
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		h.sendError(w, http.StatusBadRequest, "Search query is required", "")
		return
	}

	// Парсим параметры пагинации
	page := 1
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	pageSize := 20
	if pageSizeStr := r.URL.Query().Get("page_size"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 {
			pageSize = ps
		}
	}

	// Выполняем поиск
	result, err := h.newsService.SearchNews(r.Context(), query, page, pageSize)
	if err != nil {
		h.logger.WithError(err).WithField("query", query).Error("Failed to search news")
		h.sendError(w, http.StatusInternalServerError, "Failed to search news", "")
		return
	}

	h.sendResponse(w, http.StatusOK, Response{
		Success: true,
		Data:    result,
	})
}

// GetCategories возвращает все категории
func (h *Handler) GetCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := h.newsService.GetCategories(r.Context())
	if err != nil {
		h.logger.WithError(err).Error("Failed to get categories")
		h.sendError(w, http.StatusInternalServerError, "Failed to get categories", "")
		return
	}

	h.sendResponse(w, http.StatusOK, Response{
		Success: true,
		Data:    categories,
	})
}

// GetSources возвращает все источники новостей
func (h *Handler) GetSources(w http.ResponseWriter, r *http.Request) {
	sources, err := h.newsService.GetSources(r.Context())
	if err != nil {
		h.logger.WithError(err).Error("Failed to get sources")
		h.sendError(w, http.StatusInternalServerError, "Failed to get sources", "")
		return
	}

	h.sendResponse(w, http.StatusOK, Response{
		Success: true,
		Data:    sources,
	})
}

// GetCountries возвращает все страны
func (h *Handler) GetCountries(w http.ResponseWriter, r *http.Request) {
	countries, err := h.newsService.GetCountries(r.Context())
	if err != nil {
		h.logger.WithError(err).Error("Failed to get countries")
		h.sendError(w, http.StatusInternalServerError, "Failed to get countries", "")
		return
	}

	h.sendResponse(w, http.StatusOK, Response{
		Success: true,
		Data:    countries,
	})
}

// GetStats возвращает статистику новостей
func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.newsService.GetStats(r.Context())
	if err != nil {
		h.logger.WithError(err).Error("Failed to get stats")
		h.sendError(w, http.StatusInternalServerError, "Failed to get stats", "")
		return
	}

	h.sendResponse(w, http.StatusOK, Response{
		Success: true,
		Data:    stats,
	})
}

// ClearCache очищает кеш
func (h *Handler) ClearCache(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.sendError(w, http.StatusMethodNotAllowed, "Method not allowed", "")
		return
	}

	h.newsService.ClearCache()

	h.sendResponse(w, http.StatusOK, Response{
		Success: true,
		Message: "Cache cleared successfully",
	})
}

// parseNewsFilter парсит параметры фильтра из HTTP запроса
func (h *Handler) parseNewsFilter(r *http.Request) (models.NewsFilter, error) {
	filter := models.NewsFilter{}

	// Парсим источники
	if sourcesStr := r.URL.Query().Get("sources"); sourcesStr != "" {
		for _, idStr := range strings.Split(sourcesStr, ",") {
			if id, err := strconv.Atoi(strings.TrimSpace(idStr)); err == nil {
				filter.SourceIDs = append(filter.SourceIDs, id)
			}
		}
	}

	// Парсим категории
	if categoriesStr := r.URL.Query().Get("categories"); categoriesStr != "" {
		for _, idStr := range strings.Split(categoriesStr, ",") {
			if id, err := strconv.Atoi(strings.TrimSpace(idStr)); err == nil {
				filter.CategoryIDs = append(filter.CategoryIDs, id)
			}
		}
	}

	// Парсим страны
	if countriesStr := r.URL.Query().Get("countries"); countriesStr != "" {
		for _, idStr := range strings.Split(countriesStr, ",") {
			if id, err := strconv.Atoi(strings.TrimSpace(idStr)); err == nil {
				filter.CountryIDs = append(filter.CountryIDs, id)
			}
		}
	}

	// Парсим ключевые слова
	filter.Keywords = strings.TrimSpace(r.URL.Query().Get("keywords"))

	// Парсим даты
	if dateFromStr := r.URL.Query().Get("date_from"); dateFromStr != "" {
		if dateFrom, err := time.Parse("2006-01-02", dateFromStr); err == nil {
			filter.DateFrom = &dateFrom
		}
	}

	if dateToStr := r.URL.Query().Get("date_to"); dateToStr != "" {
		if dateTo, err := time.Parse("2006-01-02", dateToStr); err == nil {
			filter.DateTo = &dateTo
		}
	}

	// Парсим минимальную релевантность
	if minRelevanceStr := r.URL.Query().Get("min_relevance"); minRelevanceStr != "" {
		if minRelevance, err := strconv.ParseFloat(minRelevanceStr, 64); err == nil {
			filter.MinRelevance = &minRelevance
		}
	}

	// Парсим пагинацию
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			filter.Page = page
		}
	}

	if pageSizeStr := r.URL.Query().Get("page_size"); pageSizeStr != "" {
		if pageSize, err := strconv.Atoi(pageSizeStr); err == nil && pageSize > 0 {
			filter.PageSize = pageSize
		}
	}

	// Парсим сортировку
	filter.SortBy = r.URL.Query().Get("sort_by")
	filter.SortOrder = r.URL.Query().Get("sort_order")

	return filter, nil
}

// extractIDFromPath извлекает ID из пути URL
func (h *Handler) extractIDFromPath(path, prefix string) string {
	if !strings.HasPrefix(path, prefix) {
		return ""
	}

	idPart := strings.TrimPrefix(path, prefix)
	if idx := strings.Index(idPart, "/"); idx != -1 {
		idPart = idPart[:idx]
	}

	return idPart
}

// sendResponse отправляет JSON ответ
func (h *Handler) sendResponse(w http.ResponseWriter, statusCode int, response interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.WithError(err).Error("Failed to encode response")
	}
}

// sendError отправляет JSON ответ с ошибкой
func (h *Handler) sendError(w http.ResponseWriter, statusCode int, message, details string) {
	response := ErrorResponse{
		Success: false,
		Error:   message,
	}

	if details != "" {
		response.Code = details
	}

	h.sendResponse(w, statusCode, response)
}

// corsMiddleware добавляет CORS заголовки (отключен, так как CORS обрабатывается в API Gateway)
func (h *Handler) corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// CORS заголовки обрабатываются в API Gateway
		// Не добавляем CORS заголовки здесь, чтобы избежать дублирования
		next(w, r)
	}
}

// loggingMiddleware логирует HTTP запросы
func (h *Handler) loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Создаем wrapper для ResponseWriter чтобы захватить status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next(wrapped, r)

		duration := time.Since(start)

		h.logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"url":         r.URL.String(),
			"status_code": wrapped.statusCode,
			"duration":    duration,
			"remote_addr": r.RemoteAddr,
			"user_agent":  r.UserAgent(),
		}).Info("HTTP request")
	}
}

// responseWriter обертка для захвата status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// SetupRoutes настраивает маршруты HTTP сервера
func (h *Handler) SetupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", h.loggingMiddleware(h.corsMiddleware(h.HealthCheck)))

	// News endpoints
	mux.HandleFunc("/api/news", h.loggingMiddleware(h.corsMiddleware(h.GetNews)))
	mux.HandleFunc("/api/news/", h.loggingMiddleware(h.corsMiddleware(h.GetNewsById))) // с слешем для ID
	mux.HandleFunc("/api/news/latest", h.loggingMiddleware(h.corsMiddleware(h.GetLatestNews)))
	mux.HandleFunc("/api/news/trending", h.loggingMiddleware(h.corsMiddleware(h.GetTrendingNews)))
	mux.HandleFunc("/api/news/search", h.loggingMiddleware(h.corsMiddleware(h.SearchNews)))

	// Reference data endpoints
	mux.HandleFunc("/api/categories", h.loggingMiddleware(h.corsMiddleware(h.GetCategories)))
	mux.HandleFunc("/api/sources", h.loggingMiddleware(h.corsMiddleware(h.GetSources)))
	mux.HandleFunc("/api/countries", h.loggingMiddleware(h.corsMiddleware(h.GetCountries)))

	// Statistics and admin endpoints
	mux.HandleFunc("/api/stats", h.loggingMiddleware(h.corsMiddleware(h.GetStats)))
	mux.HandleFunc("/api/cache/clear", h.loggingMiddleware(h.corsMiddleware(h.ClearCache)))

	return mux
}

var startTime = time.Now() // Время запуска сервиса
