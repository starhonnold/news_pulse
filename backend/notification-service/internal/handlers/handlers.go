package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"notification-service/internal/models"
	"notification-service/internal/services"
)

// Handler представляет HTTP обработчики Notification Service
type Handler struct {
	notificationService *services.NotificationService
	websocketService    *services.WebSocketService
	logger              *logrus.Logger
}

// NewHandler создает новый обработчик
func NewHandler(
	notificationService *services.NotificationService,
	websocketService *services.WebSocketService,
	logger *logrus.Logger,
) *Handler {
	return &Handler{
		notificationService: notificationService,
		websocketService:    websocketService,
		logger:              logger,
	}
}

// CreateNotification создает новое уведомление
func (h *Handler) CreateNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.sendError(w, http.StatusMethodNotAllowed, models.ErrorCodeValidation, "Method not allowed")
		return
	}

	var req models.CreateNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, models.ErrorCodeValidation, "Invalid JSON")
		return
	}

	notification, err := h.notificationService.CreateNotification(&req)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.sendResponse(w, http.StatusCreated, models.NewSuccessResponse(notification))
}

// GetNotification получает уведомление по ID
func (h *Handler) GetNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.sendError(w, http.StatusMethodNotAllowed, models.ErrorCodeValidation, "Method not allowed")
		return
	}

	id, err := h.extractIDFromPath(r.URL.Path, "/api/notifications/")
	if err != nil {
		h.sendError(w, http.StatusBadRequest, models.ErrorCodeValidation, "Invalid notification ID")
		return
	}

	notification, err := h.notificationService.GetNotification(id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.sendResponse(w, http.StatusOK, models.NewSuccessResponse(notification))
}

// GetUserNotifications получает уведомления пользователя
func (h *Handler) GetUserNotifications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.sendError(w, http.StatusMethodNotAllowed, models.ErrorCodeValidation, "Method not allowed")
		return
	}

	userID, err := h.extractIDFromPath(r.URL.Path, "/api/users/")
	if err != nil {
		h.sendError(w, http.StatusBadRequest, models.ErrorCodeValidation, "Invalid user ID")
		return
	}

	// Парсим параметры фильтра
	filter := h.parseNotificationFilter(r)
	filter.UserID = userID

	notifications, err := h.notificationService.GetUserNotifications(userID, filter)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.sendResponse(w, http.StatusOK, models.NewPaginatedResponse(
		notifications.Notifications,
		notifications.Page,
		notifications.PageSize,
		notifications.Total,
	))
}

// MarkNotificationAsRead помечает уведомление как прочитанное
func (h *Handler) MarkNotificationAsRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		h.sendError(w, http.StatusMethodNotAllowed, models.ErrorCodeValidation, "Method not allowed")
		return
	}

	id, err := h.extractIDFromPath(r.URL.Path, "/api/notifications/")
	if err != nil {
		h.sendError(w, http.StatusBadRequest, models.ErrorCodeValidation, "Invalid notification ID")
		return
	}

	if err := h.notificationService.MarkNotificationAsRead(id); err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.sendResponse(w, http.StatusOK, models.NewSuccessResponse(map[string]interface{}{
		"message": "Notification marked as read",
	}))
}

// MarkAllNotificationsAsRead помечает все уведомления пользователя как прочитанные
func (h *Handler) MarkAllNotificationsAsRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		h.sendError(w, http.StatusMethodNotAllowed, models.ErrorCodeValidation, "Method not allowed")
		return
	}

	userID, err := h.extractIDFromPath(r.URL.Path, "/api/users/")
	if err != nil {
		h.sendError(w, http.StatusBadRequest, models.ErrorCodeValidation, "Invalid user ID")
		return
	}

	if err := h.notificationService.MarkAllNotificationsAsRead(userID); err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.sendResponse(w, http.StatusOK, models.NewSuccessResponse(map[string]interface{}{
		"message": "All notifications marked as read",
	}))
}

// DeleteNotification удаляет уведомление
func (h *Handler) DeleteNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		h.sendError(w, http.StatusMethodNotAllowed, models.ErrorCodeValidation, "Method not allowed")
		return
	}

	id, err := h.extractIDFromPath(r.URL.Path, "/api/notifications/")
	if err != nil {
		h.sendError(w, http.StatusBadRequest, models.ErrorCodeValidation, "Invalid notification ID")
		return
	}

	if err := h.notificationService.DeleteNotification(id); err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.sendResponse(w, http.StatusOK, models.NewSuccessResponse(map[string]interface{}{
		"message": "Notification deleted",
	}))
}

// GetNotificationStats возвращает статистику уведомлений пользователя
func (h *Handler) GetNotificationStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.sendError(w, http.StatusMethodNotAllowed, models.ErrorCodeValidation, "Method not allowed")
		return
	}

	userID, err := h.extractIDFromPath(r.URL.Path, "/api/users/")
	if err != nil {
		h.sendError(w, http.StatusBadRequest, models.ErrorCodeValidation, "Invalid user ID")
		return
	}

	stats, err := h.notificationService.GetNotificationStats(userID)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.sendResponse(w, http.StatusOK, models.NewSuccessResponse(stats))
}

// GetUnreadCount возвращает количество непрочитанных уведомлений
func (h *Handler) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.sendError(w, http.StatusMethodNotAllowed, models.ErrorCodeValidation, "Method not allowed")
		return
	}

	userID, err := h.extractIDFromPath(r.URL.Path, "/api/users/")
	if err != nil {
		h.sendError(w, http.StatusBadRequest, models.ErrorCodeValidation, "Invalid user ID")
		return
	}

	count, err := h.notificationService.GetUnreadCount(userID)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.sendResponse(w, http.StatusOK, models.NewSuccessResponse(map[string]interface{}{
		"unread_count": count,
	}))
}

// CreateNewsAlert создает уведомление о новости
func (h *Handler) CreateNewsAlert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.sendError(w, http.StatusMethodNotAllowed, models.ErrorCodeValidation, "Method not allowed")
		return
	}

	var req struct {
		UserID int                        `json:"user_id"`
		Data   models.NewsAlertData       `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, models.ErrorCodeValidation, "Invalid JSON")
		return
	}

	if req.UserID <= 0 {
		h.sendError(w, http.StatusBadRequest, models.ErrorCodeValidation, "Invalid user ID")
		return
	}

	if err := h.notificationService.CreateNewsAlert(req.UserID, &req.Data); err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.sendResponse(w, http.StatusCreated, models.NewSuccessResponse(map[string]interface{}{
		"message": "News alert created",
	}))
}

// CreatePulseUpdate создает уведомление об обновлении пульса
func (h *Handler) CreatePulseUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.sendError(w, http.StatusMethodNotAllowed, models.ErrorCodeValidation, "Method not allowed")
		return
	}

	var req struct {
		UserID int                         `json:"user_id"`
		Data   models.PulseUpdateData      `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, models.ErrorCodeValidation, "Invalid JSON")
		return
	}

	if req.UserID <= 0 {
		h.sendError(w, http.StatusBadRequest, models.ErrorCodeValidation, "Invalid user ID")
		return
	}

	if err := h.notificationService.CreatePulseUpdate(req.UserID, &req.Data); err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.sendResponse(w, http.StatusCreated, models.NewSuccessResponse(map[string]interface{}{
		"message": "Pulse update created",
	}))
}

// CreateSystemMessage создает системное уведомление
func (h *Handler) CreateSystemMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.sendError(w, http.StatusMethodNotAllowed, models.ErrorCodeValidation, "Method not allowed")
		return
	}

	var req struct {
		UserIDs []int                       `json:"user_ids"`
		Message string                      `json:"message"`
		Data    models.SystemMessageData    `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, models.ErrorCodeValidation, "Invalid JSON")
		return
	}

	if len(req.UserIDs) == 0 {
		h.sendError(w, http.StatusBadRequest, models.ErrorCodeValidation, "User IDs are required")
		return
	}

	if req.Message == "" {
		h.sendError(w, http.StatusBadRequest, models.ErrorCodeValidation, "Message is required")
		return
	}

	if err := h.notificationService.CreateSystemMessage(req.UserIDs, req.Message, &req.Data); err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.sendResponse(w, http.StatusCreated, models.NewSuccessResponse(map[string]interface{}{
		"message": "System message created",
	}))
}

// HealthCheck возвращает статус здоровья сервиса
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	health := models.ServiceHealth{
		Status:             models.StatusHealthy,
		Timestamp:          time.Now(),
		Version:            "1.0.0",
		Uptime:             time.Since(startTime),
		DatabaseStatus:     models.StatusHealthy, // TODO: реальная проверка БД
		WebSocketStatus:    h.websocketService.GetStatus(),
		TotalNotifications: 0, // TODO: реальная статистика
		ActiveConnections:  0, // TODO: реальная статистика
	}

	// Проверяем WebSocket соединение
	if !h.websocketService.IsConnected() {
		health.Status = models.StatusUnhealthy
		health.WebSocketStatus = models.StatusUnhealthy
	}

	statusCode := http.StatusOK
	if health.Status != models.StatusHealthy {
		statusCode = http.StatusServiceUnavailable
	}

	h.sendResponse(w, statusCode, models.NewSuccessResponse(health))
}

// GetServiceStats возвращает статистику сервиса
func (h *Handler) GetServiceStats(w http.ResponseWriter, r *http.Request) {
	stats := map[string]interface{}{
		"service": map[string]interface{}{
			"name":    "notification-service",
			"version": "1.0.0",
			"uptime":  time.Since(startTime),
		},
		"websocket": h.websocketService.GetStats(),
	}

	h.sendResponse(w, http.StatusOK, models.NewSuccessResponse(stats))
}

// TestWebSocket тестирует WebSocket соединение
func (h *Handler) TestWebSocket(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.sendError(w, http.StatusMethodNotAllowed, models.ErrorCodeValidation, "Method not allowed")
		return
	}

	if err := h.websocketService.TestConnection(); err != nil {
		h.sendError(w, http.StatusServiceUnavailable, models.ErrorCodeInternalError, err.Error())
		return
	}

	h.sendResponse(w, http.StatusOK, models.NewSuccessResponse(map[string]interface{}{
		"message": "WebSocket test successful",
	}))
}

// Вспомогательные методы

// sendResponse отправляет JSON ответ
func (h *Handler) sendResponse(w http.ResponseWriter, statusCode int, response *models.Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.WithError(err).Error("Failed to encode response")
	}
}

// sendError отправляет ошибку
func (h *Handler) sendError(w http.ResponseWriter, statusCode int, code, message string) {
	apiError := models.NewAPIError(code, message)
	response := models.NewErrorResponse(apiError)

	h.sendResponse(w, statusCode, response)
}

// handleServiceError обрабатывает ошибки сервиса
func (h *Handler) handleServiceError(w http.ResponseWriter, err error) {
	if apiErr, ok := err.(*models.APIError); ok {
		statusCode := http.StatusInternalServerError

		switch apiErr.Code {
		case models.ErrorCodeValidation:
			statusCode = http.StatusBadRequest
		case models.ErrorCodeNotFound:
			statusCode = http.StatusNotFound
		case models.ErrorCodeUnauthorized:
			statusCode = http.StatusUnauthorized
		case models.ErrorCodeRateLimitExceeded:
			statusCode = http.StatusTooManyRequests
		}

		h.sendResponse(w, statusCode, models.NewErrorResponse(apiErr))
	} else {
		h.sendError(w, http.StatusInternalServerError, models.ErrorCodeInternalError, err.Error())
	}
}

// extractIDFromPath извлекает ID из пути URL
func (h *Handler) extractIDFromPath(path, prefix string) (int, error) {
	if !strings.HasPrefix(path, prefix) {
		return 0, models.NewAPIError(models.ErrorCodeValidation, "Invalid path")
	}

	idStr := strings.TrimPrefix(path, prefix)
	idStr = strings.Split(idStr, "/")[0] // Берем первую часть после префикса

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return 0, models.NewAPIError(models.ErrorCodeValidation, "Invalid ID format")
	}

	return id, nil
}

// parseNotificationFilter парсит параметры фильтра из URL
func (h *Handler) parseNotificationFilter(r *http.Request) *models.NotificationFilter {
	filter := &models.NotificationFilter{
		Page:     1,
		PageSize: 20,
	}

	query := r.URL.Query()

	// Парсим тип уведомления
	if notificationType := query.Get("type"); notificationType != "" {
		filter.Type = notificationType
	}

	// Парсим статус прочтения
	if isReadStr := query.Get("is_read"); isReadStr != "" {
		if isReadStr == "true" {
			isRead := true
			filter.IsRead = &isRead
		} else if isReadStr == "false" {
			isRead := false
			filter.IsRead = &isRead
		}
	}

	// Парсим дату от
	if dateFromStr := query.Get("date_from"); dateFromStr != "" {
		if dateFrom, err := time.Parse(time.RFC3339, dateFromStr); err == nil {
			filter.DateFrom = &dateFrom
		}
	}

	// Парсим дату до
	if dateToStr := query.Get("date_to"); dateToStr != "" {
		if dateTo, err := time.Parse(time.RFC3339, dateToStr); err == nil {
			filter.DateTo = &dateTo
		}
	}

	// Парсим страницу
	if pageStr := query.Get("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			filter.Page = page
		}
	}

	// Парсим размер страницы
	if pageSizeStr := query.Get("page_size"); pageSizeStr != "" {
		if pageSize, err := strconv.Atoi(pageSizeStr); err == nil && pageSize > 0 && pageSize <= 100 {
			filter.PageSize = pageSize
		}
	}

	return filter
}

var startTime = time.Now() // Время запуска сервиса
