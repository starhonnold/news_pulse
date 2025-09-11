package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"api-gateway/internal/middleware"
	"api-gateway/internal/models"
	"api-gateway/internal/services"
)

// Handler представляет HTTP обработчики API Gateway
type Handler struct {
	proxyService     *services.ProxyService
	websocketService *services.WebSocketService
	authMiddleware   *middleware.AuthMiddleware
	logger           *logrus.Logger
}

// NewHandler создает новый обработчик
func NewHandler(
	proxyService *services.ProxyService,
	websocketService *services.WebSocketService,
	authMiddleware *middleware.AuthMiddleware,
	logger *logrus.Logger,
) *Handler {
	return &Handler{
		proxyService:     proxyService,
		websocketService: websocketService,
		authMiddleware:   authMiddleware,
		logger:           logger,
	}
}

// HealthCheck возвращает статус здоровья API Gateway
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	// Проверяем состояние всех микросервисов
	serviceHealth := h.proxyService.CheckAllServicesHealth(r.Context())

	// Определяем общий статус
	overallStatus := models.ServiceStatusHealthy
	for _, health := range serviceHealth {
		if health.Status != models.ServiceStatusHealthy {
			overallStatus = models.ServiceStatusUnhealthy
			break
		}
	}

	// Собираем метрики
	metrics := models.GatewayMetrics{
		TotalRequests:     0, // TODO: реализовать счетчики
		ActiveConnections: h.websocketService.GetStats()["total_connections"].(int),
		RequestsPerSecond: 0, // TODO: реализовать счетчики
		AverageLatency:    0, // TODO: реализовать счетчики
		ErrorRate:         0, // TODO: реализовать счетчики
		ServiceMetrics:    make(map[string]int64),
	}

	health := models.GatewayHealth{
		Status:    overallStatus,
		Timestamp: time.Now(),
		Version:   "1.0.0",
		Uptime:    time.Since(startTime),
		Services:  serviceHealth,
		Metrics:   metrics,
	}

	statusCode := http.StatusOK
	if overallStatus != models.ServiceStatusHealthy {
		statusCode = http.StatusServiceUnavailable
	}

	h.sendResponse(w, statusCode, models.NewSuccessResponse(health, middleware.GetRequestID(r.Context())))
}

// ProxyRequest проксирует запрос к микросервисам
func (h *Handler) ProxyRequest(w http.ResponseWriter, r *http.Request) {
	h.proxyService.ProxyRequest(w, r)
}

// HandleWebSocket обрабатывает WebSocket соединения
func (h *Handler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	h.websocketService.HandleWebSocket(w, r)
}

// Login обрабатывает аутентификацию пользователей
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.sendError(w, r, http.StatusMethodNotAllowed, models.ErrorCodeBadRequest, "Method not allowed")
		return
	}

	var loginReq models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		h.sendError(w, r, http.StatusBadRequest, models.ErrorCodeBadRequest, "Invalid JSON")
		return
	}

	if err := loginReq.Validate(); err != nil {
		if apiErr, ok := err.(*models.APIError); ok {
			h.sendAPIError(w, r, http.StatusBadRequest, apiErr)
		} else {
			h.sendError(w, r, http.StatusBadRequest, models.ErrorCodeValidationError, err.Error())
		}
		return
	}

	// TODO: Реализовать реальную аутентификацию с проверкой пароля в БД
	h.logger.WithField("username", loginReq.Username).Warn("Authentication not implemented - returning error")
	h.sendError(w, r, http.StatusNotImplemented, models.ErrorCodeInternalError, "Authentication service not implemented")
}

// RefreshToken обновляет JWT токен
func (h *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.sendError(w, r, http.StatusMethodNotAllowed, models.ErrorCodeBadRequest, "Method not allowed")
		return
	}

	var refreshReq models.RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&refreshReq); err != nil {
		h.sendError(w, r, http.StatusBadRequest, models.ErrorCodeBadRequest, "Invalid JSON")
		return
	}

	if err := refreshReq.Validate(); err != nil {
		if apiErr, ok := err.(*models.APIError); ok {
			h.sendAPIError(w, r, http.StatusBadRequest, apiErr)
		} else {
			h.sendError(w, r, http.StatusBadRequest, models.ErrorCodeValidationError, err.Error())
		}
		return
	}

	// Валидируем refresh токен
	userID, err := h.authMiddleware.ValidateRefreshToken(refreshReq.RefreshToken)
	if err != nil {
		h.sendError(w, r, http.StatusUnauthorized, models.ErrorCodeUnauthorized, "Invalid refresh token")
		return
	}

	// TODO: Получить пользователя из БД по userID
	// TODO: Реализовать реальную регистрацию
	user := models.User{
		ID:        userID,
		Username:  "user" + string(rune(userID)),
		Email:     "user" + string(rune(userID)) + "@example.com",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Генерируем новые токены
	accessToken, err := h.authMiddleware.GenerateToken(user)
	if err != nil {
		h.logger.WithError(err).Error("Failed to generate access token")
		h.sendError(w, r, http.StatusInternalServerError, models.ErrorCodeInternalError, "Failed to generate token")
		return
	}

	newRefreshToken, err := h.authMiddleware.GenerateRefreshToken(user)
	if err != nil {
		h.logger.WithError(err).Error("Failed to generate refresh token")
		h.sendError(w, r, http.StatusInternalServerError, models.ErrorCodeInternalError, "Failed to generate refresh token")
		return
	}

	authResponse := models.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresAt:    time.Now().Add(24 * time.Hour), // TODO: использовать конфиг
		User:         user,
	}

	h.logger.WithField("user_id", userID).Info("Token refreshed successfully")

	h.sendResponse(w, http.StatusOK, models.NewSuccessResponse(authResponse, middleware.GetRequestID(r.Context())))
}

// Register обрабатывает регистрацию новых пользователей
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.sendError(w, r, http.StatusMethodNotAllowed, models.ErrorCodeBadRequest, "Method not allowed")
		return
	}

	var registerReq models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&registerReq); err != nil {
		h.sendError(w, r, http.StatusBadRequest, models.ErrorCodeBadRequest, "Invalid JSON")
		return
	}

	if err := registerReq.Validate(); err != nil {
		if apiErr, ok := err.(*models.APIError); ok {
			h.sendAPIError(w, r, http.StatusBadRequest, apiErr)
		} else {
			h.sendError(w, r, http.StatusBadRequest, models.ErrorCodeValidationError, err.Error())
		}
		return
	}

	// TODO: Реализовать реальную регистрацию с сохранением в БД
	h.logger.WithFields(logrus.Fields{
		"username": registerReq.Username,
		"email":    registerReq.Email,
	}).Warn("Registration not implemented - returning error")
	h.sendError(w, r, http.StatusNotImplemented, models.ErrorCodeInternalError, "Registration service not implemented")
}

// GetStats возвращает статистику API Gateway
func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats := map[string]interface{}{
		"gateway": map[string]interface{}{
			"version": "1.0.0",
			"uptime":  time.Since(startTime),
		},
		"proxy":     h.proxyService.GetStats(),
		"websocket": h.websocketService.GetStats(),
	}

	h.sendResponse(w, http.StatusOK, models.NewSuccessResponse(stats, middleware.GetRequestID(r.Context())))
}

// NotFound обрабатывает неизвестные маршруты
func (h *Handler) NotFound(w http.ResponseWriter, r *http.Request) {
	h.sendError(w, r, http.StatusNotFound, models.ErrorCodeNotFound, "Endpoint not found")
}

// Вспомогательные методы

// sendResponse отправляет JSON ответ
func (h *Handler) sendResponse(w http.ResponseWriter, statusCode int, response *models.Response) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.WithError(err).Error("Failed to encode response")
	}
}

// sendError отправляет ошибку
func (h *Handler) sendError(w http.ResponseWriter, r *http.Request, statusCode int, code, message string) {
	requestID := middleware.GetRequestID(r.Context())
	apiError := models.NewAPIError(code, message)
	response := models.NewErrorResponse(apiError, requestID)

	h.sendResponse(w, statusCode, response)
}

// sendAPIError отправляет API ошибку
func (h *Handler) sendAPIError(w http.ResponseWriter, r *http.Request, statusCode int, apiError *models.APIError) {
	requestID := middleware.GetRequestID(r.Context())
	response := models.NewErrorResponse(apiError, requestID)

	h.sendResponse(w, statusCode, response)
}

var startTime = time.Now() // Время запуска сервиса
