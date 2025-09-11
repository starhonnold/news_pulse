package models

import (
	"fmt"
	"time"
)

// Notification представляет уведомление пользователю
type Notification struct {
	ID         int       `json:"id" db:"id"`
	UserID     int       `json:"user_id" db:"user_id"`
	Type       string    `json:"type" db:"type"`
	Title      string    `json:"title" db:"title"`
	Message    string    `json:"message" db:"message"`
	Data       string    `json:"data,omitempty" db:"data"` // JSON данные
	IsRead     bool      `json:"is_read" db:"is_read"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	ReadAt     *time.Time `json:"read_at,omitempty" db:"read_at"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty" db:"expires_at"`
}

// CreateNotificationRequest представляет запрос на создание уведомления
type CreateNotificationRequest struct {
	UserID    int                    `json:"user_id" validate:"required,min=1"`
	Type      string                 `json:"type" validate:"required"`
	Title     string                 `json:"title" validate:"required,max=200"`
	Message   string                 `json:"message" validate:"required,max=500"`
	Data      map[string]interface{} `json:"data,omitempty"`
	ExpiresAt *time.Time             `json:"expires_at,omitempty"`
}

// UpdateNotificationRequest представляет запрос на обновление уведомления
type UpdateNotificationRequest struct {
	IsRead bool `json:"is_read"`
}

// NotificationFilter представляет фильтр для уведомлений
type NotificationFilter struct {
	UserID    int       `json:"user_id,omitempty"`
	Type      string    `json:"type,omitempty"`
	IsRead    *bool     `json:"is_read,omitempty"`
	DateFrom  *time.Time `json:"date_from,omitempty"`
	DateTo    *time.Time `json:"date_to,omitempty"`
	Page      int       `json:"page"`
	PageSize  int       `json:"page_size"`
}

// NotificationStats представляет статистику уведомлений
type NotificationStats struct {
	TotalNotifications  int                    `json:"total_notifications"`
	UnreadNotifications int                    `json:"unread_notifications"`
	NotificationsByType map[string]int         `json:"notifications_by_type"`
	RecentActivity      []NotificationActivity `json:"recent_activity"`
}

// NotificationActivity представляет активность уведомлений
type NotificationActivity struct {
	Date  time.Time `json:"date"`
	Count int       `json:"count"`
}

// WebSocketNotification представляет уведомление для WebSocket
type WebSocketNotification struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
	UserID  int         `json:"user_id,omitempty"`
}

// NewsAlertData представляет данные для новостного оповещения
type NewsAlertData struct {
	NewsID      int    `json:"news_id"`
	Title       string `json:"title"`
	Summary     string `json:"summary"`
	URL         string `json:"url"`
	SourceName  string `json:"source_name"`
	Category    string `json:"category"`
	PublishedAt time.Time `json:"published_at"`
}

// PulseUpdateData представляет данные для обновления пульса
type PulseUpdateData struct {
	PulseID    int    `json:"pulse_id"`
	PulseName  string `json:"pulse_name"`
	NewsCount  int    `json:"news_count"`
	UpdateType string `json:"update_type"` // new_news, pulse_modified
}

// SystemMessageData представляет данные для системного сообщения
type SystemMessageData struct {
	MessageType string                 `json:"message_type"`
	Priority    string                 `json:"priority"` // low, medium, high, urgent
	ActionURL   string                 `json:"action_url,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// NotificationEvent представляет событие для создания уведомления
type NotificationEvent struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	UserID    int                    `json:"user_id,omitempty"`
	UserIDs   []int                  `json:"user_ids,omitempty"` // для массовых уведомлений
	Title     string                 `json:"title"`
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Priority  string                 `json:"priority,omitempty"`
	ExpiresAt *time.Time             `json:"expires_at,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

// NotificationTemplate представляет шаблон уведомления
type NotificationTemplate struct {
	Type        string            `json:"type"`
	Title       string            `json:"title"`
	Body        string            `json:"body"`
	Variables   map[string]string `json:"variables"`
	Priority    string            `json:"priority"`
	ExpiresIn   *time.Duration    `json:"expires_in,omitempty"`
}

// UserNotificationSettings представляет настройки уведомлений пользователя
type UserNotificationSettings struct {
	UserID              int  `json:"user_id" db:"user_id"`
	NewsAlertsEnabled   bool `json:"news_alerts_enabled" db:"news_alerts_enabled"`
	PulseUpdatesEnabled bool `json:"pulse_updates_enabled" db:"pulse_updates_enabled"`
	SystemMessagesEnabled bool `json:"system_messages_enabled" db:"system_messages_enabled"`
	EmailNotifications  bool `json:"email_notifications" db:"email_notifications"`
	PushNotifications   bool `json:"push_notifications" db:"push_notifications"`
	SMSNotifications    bool `json:"sms_notifications" db:"sms_notifications"`
	QuietHoursStart     *string `json:"quiet_hours_start,omitempty" db:"quiet_hours_start"`
	QuietHoursEnd       *string `json:"quiet_hours_end,omitempty" db:"quiet_hours_end"`
	CreatedAt           time.Time `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time `json:"updated_at" db:"updated_at"`
}

// PaginatedNotifications представляет список уведомлений с пагинацией
type PaginatedNotifications struct {
	Notifications []Notification `json:"notifications"`
	Total         int            `json:"total"`
	Page          int            `json:"page"`
	PageSize      int            `json:"page_size"`
	TotalPages    int            `json:"total_pages"`
}

// ServiceHealth представляет состояние сервиса
type ServiceHealth struct {
	Status           string        `json:"status"`
	Timestamp        time.Time     `json:"timestamp"`
	Version          string        `json:"version"`
	Uptime           time.Duration `json:"uptime"`
	DatabaseStatus   string        `json:"database_status"`
	WebSocketStatus  string        `json:"websocket_status"`
	TotalNotifications int         `json:"total_notifications"`
	ActiveConnections  int         `json:"active_connections"`
}

// APIError представляет ошибку API
type APIError struct {
	Code      string    `json:"code"`
	Message   string    `json:"message"`
	Details   string    `json:"details,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// Error реализует интерфейс error
func (e *APIError) Error() string {
	return e.Message
}

// Response представляет стандартный ответ API
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
}

// Константы для типов уведомлений
const (
	NotificationTypeNewsAlert     = "news_alert"
	NotificationTypePulseUpdate   = "pulse_update"
	NotificationTypeSystemMessage = "system_message"
	NotificationTypeUserMention   = "user_mention"
)

// Константы для приоритетов
const (
	PriorityLow    = "low"
	PriorityMedium = "medium"
	PriorityHigh   = "high"
	PriorityUrgent = "urgent"
)

// Константы для статусов
const (
	StatusHealthy   = "healthy"
	StatusUnhealthy = "unhealthy"
	StatusUnknown   = "unknown"
)

// Константы для кодов ошибок
const (
	ErrorCodeValidation       = "VALIDATION_ERROR"
	ErrorCodeNotFound         = "NOT_FOUND"
	ErrorCodeDatabaseError    = "DATABASE_ERROR"
	ErrorCodeInternalError    = "INTERNAL_ERROR"
	ErrorCodeUnauthorized     = "UNAUTHORIZED"
	ErrorCodeRateLimitExceeded = "RATE_LIMIT_EXCEEDED"
)

// Validation методы

// Validate проверяет валидность запроса на создание уведомления
func (r *CreateNotificationRequest) Validate() error {
	if r.UserID <= 0 {
		return &APIError{
			Code:    ErrorCodeValidation,
			Message: "User ID must be positive",
		}
	}
	
	if r.Type == "" {
		return &APIError{
			Code:    ErrorCodeValidation,
			Message: "Notification type is required",
		}
	}
	
	if len(r.Title) == 0 || len(r.Title) > 200 {
		return &APIError{
			Code:    ErrorCodeValidation,
			Message: "Title must be 1-200 characters long",
		}
	}
	
	if len(r.Message) == 0 || len(r.Message) > 500 {
		return &APIError{
			Code:    ErrorCodeValidation,
			Message: "Message must be 1-500 characters long",
		}
	}
	
	return nil
}

// Validate проверяет валидность фильтра уведомлений
func (f *NotificationFilter) Validate() error {
	if f.Page < 1 {
		f.Page = 1
	}
	
	if f.PageSize < 1 || f.PageSize > 100 {
		f.PageSize = 20
	}
	
	if f.DateFrom != nil && f.DateTo != nil && f.DateFrom.After(*f.DateTo) {
		return &APIError{
			Code:    ErrorCodeValidation,
			Message: "Date from cannot be after date to",
		}
	}
	
	return nil
}

// IsExpired проверяет, истекло ли уведомление
func (n *Notification) IsExpired() bool {
	return n.ExpiresAt != nil && time.Now().After(*n.ExpiresAt)
}

// MarkAsRead помечает уведомление как прочитанное
func (n *Notification) MarkAsRead() {
	n.IsRead = true
	now := time.Now()
	n.ReadAt = &now
}

// GetPriority возвращает приоритет уведомления на основе типа
func (n *Notification) GetPriority() string {
	switch n.Type {
	case NotificationTypeSystemMessage:
		return PriorityHigh
	case NotificationTypeNewsAlert:
		return PriorityMedium
	case NotificationTypePulseUpdate:
		return PriorityLow
	default:
		return PriorityLow
	}
}

// Вспомогательные функции

// NewAPIError создает новую API ошибку
func NewAPIError(code, message string) *APIError {
	return &APIError{
		Code:      code,
		Message:   message,
		Timestamp: time.Now(),
	}
}

// NewAPIErrorWithDetails создает новую API ошибку с деталями
func NewAPIErrorWithDetails(code, message, details string) *APIError {
	return &APIError{
		Code:      code,
		Message:   message,
		Details:   details,
		Timestamp: time.Now(),
	}
}

// NewSuccessResponse создает успешный ответ
func NewSuccessResponse(data interface{}) *Response {
	return &Response{
		Success: true,
		Data:    data,
	}
}

// NewErrorResponse создает ответ с ошибкой
func NewErrorResponse(err *APIError) *Response {
	return &Response{
		Success: false,
		Error:   err,
	}
}

// NewPaginatedResponse создает ответ с пагинацией
func NewPaginatedResponse(data interface{}, page, pageSize, total int) *Response {
	totalPages := (total + pageSize - 1) / pageSize
	
	return &Response{
		Success: true,
		Data:    data,
		Meta: map[string]interface{}{
			"page":        page,
			"page_size":   pageSize,
			"total":       total,
			"total_pages": totalPages,
		},
	}
}

// NewNotificationEvent создает новое событие уведомления
func NewNotificationEvent(eventType string, userID int, title, message string) *NotificationEvent {
	return &NotificationEvent{
		ID:        generateEventID(),
		Type:      eventType,
		UserID:    userID,
		Title:     title,
		Message:   message,
		CreatedAt: time.Now(),
	}
}

// NewWebSocketNotification создает новое WebSocket уведомление
func NewWebSocketNotification(notificationType string, payload interface{}, userID int) *WebSocketNotification {
	return &WebSocketNotification{
		Type:    notificationType,
		Payload: payload,
		UserID:  userID,
	}
}

// generateEventID генерирует уникальный ID события
func generateEventID() string {
	return fmt.Sprintf("evt_%d_%d", time.Now().UnixNano(), time.Now().Unix()%1000)
}

// IsValidNotificationType проверяет валидность типа уведомления
func IsValidNotificationType(notificationType string) bool {
	validTypes := []string{
		NotificationTypeNewsAlert,
		NotificationTypePulseUpdate,
		NotificationTypeSystemMessage,
		NotificationTypeUserMention,
	}
	
	for _, validType := range validTypes {
		if notificationType == validType {
			return true
		}
	}
	return false
}

// IsValidPriority проверяет валидность приоритета
func IsValidPriority(priority string) bool {
	validPriorities := []string{
		PriorityLow,
		PriorityMedium,
		PriorityHigh,
		PriorityUrgent,
	}
	
	for _, validPriority := range validPriorities {
		if priority == validPriority {
			return true
		}
	}
	return false
}
