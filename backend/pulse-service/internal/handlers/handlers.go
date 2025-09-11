package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"pulse-service/internal/models"
	"pulse-service/internal/services"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

// Handler представляет HTTP обработчики
type Handler struct {
	pulseService *services.PulseService
	logger       *logrus.Logger
}

// NewHandler создает новый обработчик
func NewHandler(pulseService *services.PulseService, logger *logrus.Logger) *Handler {
	return &Handler{
		pulseService: pulseService,
		logger:       logger,
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
		"service":   "pulse-service",
		"version":   "1.0.0",
		"uptime":    time.Since(startTime),
	}

	// Добавляем статистику кеша
	health["cache"] = h.pulseService.GetCacheStats()

	h.sendResponse(w, http.StatusOK, Response{
		Success: true,
		Data:    health,
	})
}

// CreatePulse создает новый пульс
func (h *Handler) CreatePulse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.sendError(w, http.StatusMethodNotAllowed, "Method not allowed", "")
		return
	}

	// Получаем ID пользователя (в реальном приложении из JWT токена)
	userID := h.getUserIDFromRequest(r)
	if userID == "" {
		h.sendError(w, http.StatusUnauthorized, "User ID is required", "")
		return
	}

	// Парсим запрос
	var req models.PulseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid JSON", err.Error())
		return
	}

	// Создаем пульс
	pulse, err := h.pulseService.CreatePulse(r.Context(), userID, req)
	if err != nil {
		h.logger.WithError(err).WithField("user_id", userID).Error("Failed to create pulse")
		if strings.Contains(err.Error(), "invalid") || strings.Contains(err.Error(), "maximum") {
			h.sendError(w, http.StatusBadRequest, err.Error(), "")
		} else {
			h.sendError(w, http.StatusInternalServerError, "Failed to create pulse", "")
		}
		return
	}

	h.sendResponse(w, http.StatusCreated, Response{
		Success: true,
		Data:    pulse,
		Message: "Pulse created successfully",
	})
}

// GetPulses возвращает список пульсов пользователя
func (h *Handler) GetPulses(w http.ResponseWriter, r *http.Request) {
	// Получаем ID пользователя
	userID := h.getUserIDFromRequest(r)
	if userID == "" {
		h.sendError(w, http.StatusUnauthorized, "User ID is required", "")
		return
	}

	// Парсим параметры фильтра
	filter, err := h.parsePulseFilter(r)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid filter parameters", err.Error())
		return
	}

	// Получаем пульсы
	pulses, err := h.pulseService.GetUserPulses(r.Context(), userID, filter)
	if err != nil {
		h.logger.WithError(err).WithField("user_id", userID).Error("Failed to get user pulses")
		h.sendError(w, http.StatusInternalServerError, "Failed to get pulses", "")
		return
	}

	h.sendResponse(w, http.StatusOK, Response{
		Success: true,
		Data:    pulses,
	})
}

// GetPulseById возвращает пульс по ID
func (h *Handler) GetPulseById(w http.ResponseWriter, r *http.Request) {
	// Получаем ID пользователя
	userID := h.getUserIDFromRequest(r)
	if userID == "" {
		h.sendError(w, http.StatusUnauthorized, "User ID is required", "")
		return
	}

	// Извлекаем ID пульса из URL
	pulseID := h.extractIDFromPath(r.URL.Path, "/api/pulses/")
	if pulseID == "" {
		h.sendError(w, http.StatusBadRequest, "Pulse ID is required", "")
		return
	}

	// Получаем пульс
	pulse, err := h.pulseService.GetPulseByID(r.Context(), pulseID, userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "access denied") {
			h.sendError(w, http.StatusNotFound, "Pulse not found", "")
		} else {
			h.logger.WithError(err).WithFields(logrus.Fields{
				"pulse_id": pulseID,
				"user_id":  userID,
			}).Error("Failed to get pulse")
			h.sendError(w, http.StatusInternalServerError, "Failed to get pulse", "")
		}
		return
	}

	h.sendResponse(w, http.StatusOK, Response{
		Success: true,
		Data:    pulse,
	})
}

// GetDefaultPulse возвращает дефолтный пульс пользователя
func (h *Handler) GetDefaultPulse(w http.ResponseWriter, r *http.Request) {
	// Получаем ID пользователя
	userID := h.getUserIDFromRequest(r)
	if userID == "" {
		h.sendError(w, http.StatusUnauthorized, "User ID is required", "")
		return
	}

	// Получаем дефолтный пульс
	pulse, err := h.pulseService.GetDefaultPulse(r.Context(), userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.sendError(w, http.StatusNotFound, "No default pulse found", "")
		} else {
			h.logger.WithError(err).WithField("user_id", userID).Error("Failed to get default pulse")
			h.sendError(w, http.StatusInternalServerError, "Failed to get default pulse", "")
		}
		return
	}

	h.sendResponse(w, http.StatusOK, Response{
		Success: true,
		Data:    pulse,
	})
}

// UpdatePulse обновляет пульс
func (h *Handler) UpdatePulse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		h.sendError(w, http.StatusMethodNotAllowed, "Method not allowed", "")
		return
	}

	// Получаем ID пользователя
	userID := h.getUserIDFromRequest(r)
	if userID == "" {
		h.sendError(w, http.StatusUnauthorized, "User ID is required", "")
		return
	}

	// Извлекаем ID пульса из URL
	pulseID := h.extractIDFromPath(r.URL.Path, "/api/pulses/")
	if pulseID == "" {
		h.sendError(w, http.StatusBadRequest, "Pulse ID is required", "")
		return
	}

	// Парсим запрос
	var req models.PulseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid JSON", err.Error())
		return
	}

	// Обновляем пульс
	pulse, err := h.pulseService.UpdatePulse(r.Context(), pulseID, userID, req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "access denied") {
			h.sendError(w, http.StatusNotFound, "Pulse not found", "")
		} else if strings.Contains(err.Error(), "invalid") {
			h.sendError(w, http.StatusBadRequest, err.Error(), "")
		} else {
			h.logger.WithError(err).WithFields(logrus.Fields{
				"pulse_id": pulseID,
				"user_id":  userID,
			}).Error("Failed to update pulse")
			h.sendError(w, http.StatusInternalServerError, "Failed to update pulse", "")
		}
		return
	}

	h.sendResponse(w, http.StatusOK, Response{
		Success: true,
		Data:    pulse,
		Message: "Pulse updated successfully",
	})
}

// DeletePulse удаляет пульс
func (h *Handler) DeletePulse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		h.sendError(w, http.StatusMethodNotAllowed, "Method not allowed", "")
		return
	}

	// Получаем ID пользователя
	userID := h.getUserIDFromRequest(r)
	if userID == "" {
		h.sendError(w, http.StatusUnauthorized, "User ID is required", "")
		return
	}

	// Извлекаем ID пульса из URL
	pulseID := h.extractIDFromPath(r.URL.Path, "/api/pulses/")
	if pulseID == "" {
		h.sendError(w, http.StatusBadRequest, "Pulse ID is required", "")
		return
	}

	// Удаляем пульс
	if err := h.pulseService.DeletePulse(r.Context(), pulseID, userID); err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "access denied") {
			h.sendError(w, http.StatusNotFound, "Pulse not found", "")
		} else {
			h.logger.WithError(err).WithFields(logrus.Fields{
				"pulse_id": pulseID,
				"user_id":  userID,
			}).Error("Failed to delete pulse")
			h.sendError(w, http.StatusInternalServerError, "Failed to delete pulse", "")
		}
		return
	}

	h.sendResponse(w, http.StatusOK, Response{
		Success: true,
		Message: "Pulse deleted successfully",
	})
}

// GetPersonalizedFeed возвращает персонализированную ленту новостей
func (h *Handler) GetPersonalizedFeed(w http.ResponseWriter, r *http.Request) {
	// Получаем ID пользователя
	userID := h.getUserIDFromRequest(r)
	if userID == "" {
		h.sendError(w, http.StatusUnauthorized, "User ID is required", "")
		return
	}

	// Извлекаем ID пульса из URL
	pulseID := h.extractIDFromPath(r.URL.Path, "/api/pulses/")
	if pulseID == "" {
		h.sendError(w, http.StatusBadRequest, "Pulse ID is required", "")
		return
	}

	// Парсим параметры запроса ленты
	feedReq, err := h.parseFeedRequest(r, pulseID)
	if err != nil {
		h.sendError(w, http.StatusBadRequest, "Invalid feed parameters", err.Error())
		return
	}

	// Получаем персонализированную ленту
	feed, err := h.pulseService.GetPersonalizedFeed(r.Context(), pulseID, userID, feedReq)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.sendError(w, http.StatusNotFound, "Pulse not found", "")
		} else if strings.Contains(err.Error(), "invalid") {
			h.sendError(w, http.StatusBadRequest, err.Error(), "")
		} else {
			h.logger.WithError(err).WithFields(logrus.Fields{
				"pulse_id": pulseID,
				"user_id":  userID,
			}).Error("Failed to get personalized feed")
			h.sendError(w, http.StatusInternalServerError, "Failed to get personalized feed", "")
		}
		return
	}

	h.sendResponse(w, http.StatusOK, Response{
		Success: true,
		Data:    feed,
	})
}

// GetLatestFeedNews возвращает последние новости ленты
func (h *Handler) GetLatestFeedNews(w http.ResponseWriter, r *http.Request) {
	// Получаем ID пользователя
	userID := h.getUserIDFromRequest(r)
	if userID == "" {
		h.sendError(w, http.StatusUnauthorized, "User ID is required", "")
		return
	}

	// Извлекаем ID пульса из URL
	pulseID := h.extractIDFromPath(r.URL.Path, "/api/pulses/")
	if pulseID == "" {
		h.sendError(w, http.StatusBadRequest, "Pulse ID is required", "")
		return
	}

	// Парсим лимит
	limit := 20
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	// Получаем последние новости
	news, err := h.pulseService.GetLatestFeedNews(r.Context(), pulseID, userID, limit)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.sendError(w, http.StatusNotFound, "Pulse not found", "")
		} else {
			h.logger.WithError(err).WithFields(logrus.Fields{
				"pulse_id": pulseID,
				"user_id":  userID,
			}).Error("Failed to get latest feed news")
			h.sendError(w, http.StatusInternalServerError, "Failed to get latest news", "")
		}
		return
	}

	h.sendResponse(w, http.StatusOK, Response{
		Success: true,
		Data:    news,
	})
}

// GetTrendingFeedNews возвращает трендовые новости ленты
func (h *Handler) GetTrendingFeedNews(w http.ResponseWriter, r *http.Request) {
	// Получаем ID пользователя
	userID := h.getUserIDFromRequest(r)
	if userID == "" {
		h.sendError(w, http.StatusUnauthorized, "User ID is required", "")
		return
	}

	// Извлекаем ID пульса из URL
	pulseID := h.extractIDFromPath(r.URL.Path, "/api/pulses/")
	if pulseID == "" {
		h.sendError(w, http.StatusBadRequest, "Pulse ID is required", "")
		return
	}

	// Парсим лимит
	limit := 20
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	// Получаем трендовые новости
	news, err := h.pulseService.GetTrendingFeedNews(r.Context(), pulseID, userID, limit)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.sendError(w, http.StatusNotFound, "Pulse not found", "")
		} else {
			h.logger.WithError(err).WithFields(logrus.Fields{
				"pulse_id": pulseID,
				"user_id":  userID,
			}).Error("Failed to get trending feed news")
			h.sendError(w, http.StatusInternalServerError, "Failed to get trending news", "")
		}
		return
	}

	h.sendResponse(w, http.StatusOK, Response{
		Success: true,
		Data:    news,
	})
}

// GetPulseNews возвращает новости для пульса
func (h *Handler) GetPulseNews(w http.ResponseWriter, r *http.Request) {
	// Получаем ID пользователя
	userID := h.getUserIDFromRequest(r)
	if userID == "" {
		h.sendError(w, http.StatusUnauthorized, "User ID is required", "")
		return
	}

	// Извлекаем ID пульса из URL
	pulseID := h.extractIDFromPath(r.URL.Path, "/api/pulses/")
	if pulseID == "" {
		h.sendError(w, http.StatusBadRequest, "Pulse ID is required", "")
		return
	}

	// Парсим лимит
	limit := 20
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	// Получаем новости пульса
	news, err := h.getPulseNewsFromDB(r.Context(), pulseID, limit)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.sendError(w, http.StatusNotFound, "Pulse not found", "")
		} else {
			h.logger.WithError(err).WithFields(logrus.Fields{
				"pulse_id": pulseID,
				"user_id":  userID,
			}).Error("Failed to get pulse news")
			h.sendError(w, http.StatusInternalServerError, "Failed to get pulse news", "")
		}
		return
	}

	// Если новостей нет, возвращаем пустой массив
	if len(news) == 0 {
		h.sendResponse(w, http.StatusOK, Response{
			Success: true,
			Data:    []models.PersonalizedNews{},
		})
		return
	}

	h.sendResponse(w, http.StatusOK, Response{
		Success: true,
		Data:    news,
	})
}

// ClearCache очищает кеш
func (h *Handler) ClearCache(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.sendError(w, http.StatusMethodNotAllowed, "Method not allowed", "")
		return
	}

	h.pulseService.ClearCache()

	h.sendResponse(w, http.StatusOK, Response{
		Success: true,
		Message: "Cache cleared successfully",
	})
}

// Вспомогательные методы

// getUserIDFromRequest извлекает ID пользователя из запроса
func (h *Handler) getUserIDFromRequest(r *http.Request) string {
	// В реальном приложении ID пользователя извлекается из JWT токена
	// Для демонстрации используем заголовок X-User-ID
	userIDStr := r.Header.Get("X-User-ID")
	if userIDStr == "" {
		// Для разработки возвращаем дефолтный User ID
		return "00000000-0000-0000-0000-000000000001"
	}

	return userIDStr
}

// parsePulseFilter парсит параметры фильтра пульсов из HTTP запроса
func (h *Handler) parsePulseFilter(r *http.Request) (models.PulseFilter, error) {
	filter := models.PulseFilter{}

	// Парсим активность
	if isActiveStr := r.URL.Query().Get("is_active"); isActiveStr != "" {
		isActive := isActiveStr == "true"
		filter.IsActive = &isActive
	}

	// Парсим дефолтность
	if isDefaultStr := r.URL.Query().Get("is_default"); isDefaultStr != "" {
		isDefault := isDefaultStr == "true"
		filter.IsDefault = &isDefault
	}

	// Парсим ключевые слова
	filter.Keywords = strings.TrimSpace(r.URL.Query().Get("keywords"))

	// Парсим даты
	if createdFromStr := r.URL.Query().Get("created_from"); createdFromStr != "" {
		if createdFrom, err := time.Parse("2006-01-02", createdFromStr); err == nil {
			filter.CreatedFrom = &createdFrom
		}
	}

	if createdToStr := r.URL.Query().Get("created_to"); createdToStr != "" {
		if createdTo, err := time.Parse("2006-01-02", createdToStr); err == nil {
			filter.CreatedTo = &createdTo
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

// parseFeedRequest парсит параметры запроса персонализированной ленты
func (h *Handler) parseFeedRequest(r *http.Request, pulseID string) (models.FeedRequest, error) {
	req := models.FeedRequest{
		PulseID: pulseID,
	}

	// Парсим пагинацию
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			req.Page = page
		}
	}

	if pageSizeStr := r.URL.Query().Get("page_size"); pageSizeStr != "" {
		if pageSize, err := strconv.Atoi(pageSizeStr); err == nil && pageSize > 0 {
			req.PageSize = pageSize
		}
	}

	// Парсим даты
	if dateFromStr := r.URL.Query().Get("date_from"); dateFromStr != "" {
		if dateFrom, err := time.Parse("2006-01-02", dateFromStr); err == nil {
			req.DateFrom = &dateFrom
		}
	}

	if dateToStr := r.URL.Query().Get("date_to"); dateToStr != "" {
		if dateTo, err := time.Parse("2006-01-02", dateToStr); err == nil {
			req.DateTo = &dateTo
		}
	}

	// Парсим минимальный скор
	if minScoreStr := r.URL.Query().Get("min_score"); minScoreStr != "" {
		if minScore, err := strconv.ParseFloat(minScoreStr, 64); err == nil {
			req.MinScore = &minScore
		}
	}

	// Парсим сортировку
	req.SortBy = r.URL.Query().Get("sort_by")
	req.SortOrder = r.URL.Query().Get("sort_order")

	return req, nil
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
			"user_id":     h.getUserIDFromRequest(r),
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

	// Pulse endpoints
	mux.HandleFunc("/api/pulses", h.loggingMiddleware(h.corsMiddleware(h.handlePulsesRoute)))
	mux.HandleFunc("/api/pulses/default", h.loggingMiddleware(h.corsMiddleware(h.GetDefaultPulse)))

	// Special route for pulse news - must be before general pulse route
	mux.HandleFunc("/api/pulses/", h.loggingMiddleware(h.corsMiddleware(h.handlePulseNewsRoute)))

	// Administrative endpoints
	mux.HandleFunc("/api/cache/clear", h.loggingMiddleware(h.corsMiddleware(h.ClearCache)))

	return mux
}

// handlePulsesRoute обрабатывает маршрут /api/pulses
func (h *Handler) handlePulsesRoute(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.GetPulses(w, r)
	case http.MethodPost:
		h.CreatePulse(w, r)
	default:
		h.sendError(w, http.StatusMethodNotAllowed, "Method not allowed", "")
	}
}

// handlePulseNewsRoute обрабатывает маршрут /api/pulses/{id}/news
func (h *Handler) handlePulseNewsRoute(w http.ResponseWriter, r *http.Request) {
	h.logger.WithField("path", r.URL.Path).WithField("method", r.Method).Info("Handling pulse news route")

	// Проверяем, что это запрос новостей пульса
	if strings.HasSuffix(r.URL.Path, "/news") {
		h.logger.Info("Routing to GetPulseNews for path: " + r.URL.Path)
		h.GetPulseNews(w, r)
		return
	}

	// Если это не запрос новостей, передаем в основной обработчик
	h.handlePulseByIdRoute(w, r)
}

// handlePulseByIdRoute обрабатывает маршрут /api/pulses/{id}
func (h *Handler) handlePulseByIdRoute(w http.ResponseWriter, r *http.Request) {
	h.logger.WithField("path", r.URL.Path).WithField("method", r.Method).Info("Handling pulse by ID route")

	// Проверяем, не является ли это запросом к персонализированной ленте
	if strings.HasSuffix(r.URL.Path, "/feed") {
		h.logger.Debug("Routing to GetPersonalizedFeed")
		h.GetPersonalizedFeed(w, r)
		return
	}

	if strings.HasSuffix(r.URL.Path, "/feed/latest") {
		h.logger.Debug("Routing to GetLatestFeedNews")
		h.GetLatestFeedNews(w, r)
		return
	}

	if strings.HasSuffix(r.URL.Path, "/feed/trending") {
		h.logger.Debug("Routing to GetTrendingFeedNews")
		h.GetTrendingFeedNews(w, r)
		return
	}

	// Обрабатываем запрос новостей пульса
	if strings.HasSuffix(r.URL.Path, "/news") {
		h.logger.Info("Routing to GetPulseNews for path: " + r.URL.Path)
		h.GetPulseNews(w, r)
		return
	}

	// Обрабатываем запрос обновления пульса
	if strings.HasSuffix(r.URL.Path, "/refresh") {
		h.logger.Debug("Routing to RefreshPulse")
		h.RefreshPulse(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.GetPulseById(w, r)
	case http.MethodPut:
		h.UpdatePulse(w, r)
	case http.MethodDelete:
		h.DeletePulse(w, r)
	default:
		h.sendError(w, http.StatusMethodNotAllowed, "Method not allowed", "")
	}
}

// getPulseNewsFromDB получает новости пульса из базы данных
func (h *Handler) getPulseNewsFromDB(ctx context.Context, pulseID string, limit int) ([]models.PersonalizedNews, error) {
	// SQL запрос для получения новостей пульса
	query := `
		SELECT 
			n.id,
			n.title,
			n.description,
			n.content,
			n.url,
			n.image_url,
			n.author,
			n.source_id,
			n.category_id,
			n.published_at,
			n.relevance_score,
			n.view_count,
			ns.name as source_name,
			ns.domain as source_domain,
			ns.logo_url as source_logo_url,
			c.name as category_name,
			c.slug as category_slug,
			c.color as category_color,
			c.icon as category_icon,
			pn.match_reason,
			pn.relevance_score as pulse_relevance_score,
			COALESCE(
				ARRAY_AGG(t.name ORDER BY t.name) FILTER (WHERE t.name IS NOT NULL),
				ARRAY[]::text[]
			) as tags
		FROM pulse_news pn
		JOIN news n ON n.id = pn.news_id
		JOIN news_sources ns ON ns.id = n.source_id
		LEFT JOIN categories c ON c.id = n.category_id
		LEFT JOIN news_tags nt ON nt.news_id = n.id
		LEFT JOIN tags t ON t.id = nt.tag_id
		WHERE pn.pulse_id = $1::uuid
		AND n.is_active = true
		GROUP BY n.id, n.title, n.description, n.content, n.url, n.image_url, n.author, 
				 n.source_id, n.category_id, n.published_at, n.relevance_score, 
				 n.view_count, ns.name, ns.domain, ns.logo_url, c.name, c.slug, 
				 c.color, c.icon, pn.match_reason, pn.relevance_score
		ORDER BY pn.relevance_score DESC, n.published_at DESC
		LIMIT $2
	`

	rows, err := h.pulseService.GetDB().QueryContext(ctx, query, pulseID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query pulse news: %w", err)
	}
	defer rows.Close()

	var newsList []models.PersonalizedNews
	for rows.Next() {
		var news models.PersonalizedNews
		var categoryName, categorySlug, categoryColor sql.NullString
		var matchReason string
		var pulseRelevanceScore float64
		var tags []string

		err := rows.Scan(
			&news.ID, &news.Title, &news.Description, &news.Content, &news.URL,
			&news.ImageURL, &news.Author, &news.SourceID, &news.CategoryID,
			&news.PublishedAt, &news.RelevanceScore, &news.ViewCount,
			&news.SourceName, &news.SourceDomain, &news.SourceLogoURL,
			&categoryName, &categorySlug, &categoryColor, &news.CategoryIcon,
			&matchReason, &pulseRelevanceScore, pq.Array(&tags),
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan pulse news: %w", err)
		}

		// Заполняем категорию если есть
		if categoryName.Valid {
			news.CategoryName = categoryName.String
			news.CategorySlug = categorySlug.String
			news.CategoryColor = categoryColor.String
		}

		// Устанавливаем теги
		news.Tags = tags

		// Устанавливаем причину попадания в ленту
		news.MatchReason = matchReason

		// Используем релевантность из pulse_news
		news.RelevanceScore = pulseRelevanceScore

		// Вычисляем персональный скор
		news.CalculatePersonalScore()

		newsList = append(newsList, news)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return newsList, nil
}

// RefreshPulse обновляет новости пульса
func (h *Handler) RefreshPulse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.sendError(w, http.StatusMethodNotAllowed, "Method not allowed", "")
		return
	}

	// Получаем ID пользователя
	userID := h.getUserIDFromRequest(r)
	if userID == "" {
		h.sendError(w, http.StatusUnauthorized, "User ID is required", "")
		return
	}

	// Извлекаем ID пульса из URL
	pulseID := h.extractIDFromPath(r.URL.Path, "/api/pulses/")
	if pulseID == "" {
		h.sendError(w, http.StatusBadRequest, "Pulse ID is required", "")
		return
	}

	// Проверяем, что пульс принадлежит пользователю
	pulse, err := h.pulseService.GetPulseByID(r.Context(), pulseID, userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.sendError(w, http.StatusNotFound, "Pulse not found", "")
		} else {
			h.logger.WithError(err).WithFields(logrus.Fields{
				"pulse_id": pulseID,
				"user_id":  userID,
			}).Error("Failed to get pulse")
			h.sendError(w, http.StatusInternalServerError, "Failed to refresh pulse", "")
		}
		return
	}

	// Обновляем время последнего обновления
	timeoutCtx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	err = h.pulseService.UpdateLastRefreshed(timeoutCtx, pulseID)
	if err != nil {
		h.logger.WithError(err).WithFields(logrus.Fields{
			"pulse_id": pulseID,
			"user_id":  userID,
		}).Warn("Failed to update last refreshed time")
	}

	// Собираем новости для пульса
	err = h.pulseService.CollectPulseNews(timeoutCtx, pulseID)
	if err != nil {
		h.logger.WithError(err).WithFields(logrus.Fields{
			"pulse_id": pulseID,
			"user_id":  userID,
		}).Warn("Failed to collect pulse news")
	}

	h.logger.WithFields(logrus.Fields{
		"pulse_id": pulseID,
		"user_id":  userID,
		"name":     pulse.Name,
	}).Info("Pulse refreshed")

	h.sendResponse(w, http.StatusOK, Response{
		Success: true,
		Message: "Pulse refreshed successfully",
	})
}

var startTime = time.Now() // Время запуска сервиса
