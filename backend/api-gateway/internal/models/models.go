package models

import (
	"time"
)

// User представляет пользователя системы
type User struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// LoginRequest представляет запрос на авторизацию
type LoginRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Password string `json:"password" validate:"required,min=6,max=100"`
}

// RegisterRequest представляет запрос на регистрацию
type RegisterRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6,max=100"`
}

// RefreshTokenRequest представляет запрос на обновление токена
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// AuthResponse представляет ответ после успешной авторизации
type AuthResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	User         User      `json:"user"`
}

// JWTClaims представляет claims JWT токена
type JWTClaims struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	IssuedAt int64  `json:"iat"`
	ExpiresAt int64 `json:"exp"`
}

// IsValid проверяет валидность claims
func (c *JWTClaims) IsValid() bool {
	return c.UserID > 0 && 
		   c.Username != "" && 
		   c.ExpiresAt > time.Now().Unix()
}

// IsExpired проверяет, истек ли токен
func (c *JWTClaims) IsExpired() bool {
	return c.ExpiresAt <= time.Now().Unix()
}

// ProxyRequest представляет запрос для проксирования
type ProxyRequest struct {
	Method      string            `json:"method"`
	Path        string            `json:"path"`
	Headers     map[string]string `json:"headers"`
	Body        []byte            `json:"body,omitempty"`
	UserID      int               `json:"user_id,omitempty"`
	RequestID   string            `json:"request_id"`
	ServiceName string            `json:"service_name"`
}

// ProxyResponse представляет ответ от микросервиса
type ProxyResponse struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       []byte            `json:"body"`
	Duration   time.Duration     `json:"duration"`
	Error      string            `json:"error,omitempty"`
}

// ServiceHealth представляет состояние микросервиса
type ServiceHealth struct {
	Name         string        `json:"name"`
	URL          string        `json:"url"`
	Status       string        `json:"status"` // healthy, unhealthy, unknown
	ResponseTime time.Duration `json:"response_time"`
	LastCheck    time.Time     `json:"last_check"`
	Error        string        `json:"error,omitempty"`
}

// GatewayHealth представляет общее состояние шлюза
type GatewayHealth struct {
	Status       string                   `json:"status"`
	Timestamp    time.Time                `json:"timestamp"`
	Version      string                   `json:"version"`
	Uptime       time.Duration            `json:"uptime"`
	Services     map[string]ServiceHealth `json:"services"`
	Metrics      GatewayMetrics           `json:"metrics"`
}

// GatewayMetrics представляет метрики шлюза
type GatewayMetrics struct {
	TotalRequests     int64             `json:"total_requests"`
	ActiveConnections int               `json:"active_connections"`
	RequestsPerSecond float64           `json:"requests_per_second"`
	AverageLatency    time.Duration     `json:"average_latency"`
	ErrorRate         float64           `json:"error_rate"`
	ServiceMetrics    map[string]int64  `json:"service_metrics"`
}

// WebSocketMessage представляет сообщение WebSocket
type WebSocketMessage struct {
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
	UserID    int                    `json:"user_id,omitempty"`
}

// WebSocketConnection представляет WebSocket соединение
type WebSocketConnection struct {
	ID       string    `json:"id"`
	UserID   int       `json:"user_id"`
	ConnectedAt time.Time `json:"connected_at"`
	LastPing    time.Time `json:"last_ping"`
	Active      bool      `json:"active"`
}

// RateLimitInfo представляет информацию о rate limiting
type RateLimitInfo struct {
	Limit     int           `json:"limit"`
	Remaining int           `json:"remaining"`
	Reset     time.Time     `json:"reset"`
	RetryAfter time.Duration `json:"retry_after,omitempty"`
}

// APIError представляет стандартную ошибку API
type APIError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Details   string `json:"details,omitempty"`
	RequestID string `json:"request_id,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// Error реализует интерфейс error
func (e *APIError) Error() string {
	return e.Message
}

// Response представляет стандартный ответ API
type Response struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     *APIError   `json:"error,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// RequestContext представляет контекст запроса
type RequestContext struct {
	RequestID string
	UserID    int
	Username  string
	StartTime time.Time
	Method    string
	Path      string
	UserAgent string
	IP        string
}

// GetDuration возвращает время выполнения запроса
func (rc *RequestContext) GetDuration() time.Duration {
	return time.Since(rc.StartTime)
}

// Validation методы

// Validate проверяет валидность запроса на логин
func (r *LoginRequest) Validate() error {
	if len(r.Username) < 3 || len(r.Username) > 50 {
		return &APIError{
			Code:    "INVALID_USERNAME",
			Message: "Username must be 3-50 characters long",
		}
	}
	
	if len(r.Password) < 6 || len(r.Password) > 100 {
		return &APIError{
			Code:    "INVALID_PASSWORD", 
			Message: "Password must be 6-100 characters long",
		}
	}
	
	return nil
}

// Validate проверяет валидность запроса на регистрацию
func (r *RegisterRequest) Validate() error {
	if len(r.Username) < 3 || len(r.Username) > 50 {
		return &APIError{
			Code:    "INVALID_USERNAME",
			Message: "Username must be 3-50 characters long",
		}
	}
	
	if len(r.Email) < 5 || len(r.Email) > 100 {
		return &APIError{
			Code:    "INVALID_EMAIL",
			Message: "Email must be 5-100 characters long",
		}
	}
	
	if len(r.Password) < 6 || len(r.Password) > 100 {
		return &APIError{
			Code:    "INVALID_PASSWORD",
			Message: "Password must be 6-100 characters long",
		}
	}
	
	return nil
}

// Validate проверяет валидность запроса на обновление токена
func (r *RefreshTokenRequest) Validate() error {
	if r.RefreshToken == "" {
		return &APIError{
			Code:    "MISSING_REFRESH_TOKEN",
			Message: "Refresh token is required",
		}
	}
	
	return nil
}

// Константы для типов WebSocket сообщений
const (
	WSMessageTypeNewsUpdate    = "news_update"
	WSMessageTypePulseUpdate   = "pulse_update"
	WSMessageTypeSystemAlert   = "system_alert"
	WSMessageTypeUserNotification = "user_notification"
	WSMessageTypePing          = "ping"
	WSMessageTypePong          = "pong"
	WSMessageTypeError         = "error"
)

// Константы для статусов сервисов
const (
	ServiceStatusHealthy   = "healthy"
	ServiceStatusUnhealthy = "unhealthy"
	ServiceStatusUnknown   = "unknown"
)

// Константы для кодов ошибок
const (
	ErrorCodeUnauthorized      = "UNAUTHORIZED"
	ErrorCodeForbidden         = "FORBIDDEN"
	ErrorCodeRateLimitExceeded = "RATE_LIMIT_EXCEEDED"
	ErrorCodeServiceUnavailable = "SERVICE_UNAVAILABLE"
	ErrorCodeInternalError     = "INTERNAL_ERROR"
	ErrorCodeBadRequest        = "BAD_REQUEST"
	ErrorCodeNotFound          = "NOT_FOUND"
	ErrorCodeValidationError   = "VALIDATION_ERROR"
)

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
func NewSuccessResponse(data interface{}, requestID string) *Response {
	return &Response{
		Success:   true,
		Data:      data,
		RequestID: requestID,
		Timestamp: time.Now(),
	}
}

// NewErrorResponse создает ответ с ошибкой
func NewErrorResponse(err *APIError, requestID string) *Response {
	if err.RequestID == "" {
		err.RequestID = requestID
	}
	
	return &Response{
		Success:   false,
		Error:     err,
		RequestID: requestID,
		Timestamp: time.Now(),
	}
}

// NewWebSocketMessage создает новое WebSocket сообщение
func NewWebSocketMessage(msgType string, data map[string]interface{}, userID int) *WebSocketMessage {
	return &WebSocketMessage{
		Type:      msgType,
		Data:      data,
		Timestamp: time.Now(),
		UserID:    userID,
	}
}

// IsValidMessageType проверяет, является ли тип сообщения валидным
func IsValidMessageType(msgType string) bool {
	validTypes := []string{
		WSMessageTypeNewsUpdate,
		WSMessageTypePulseUpdate,
		WSMessageTypeSystemAlert,
		WSMessageTypeUserNotification,
		WSMessageTypePing,
		WSMessageTypePong,
		WSMessageTypeError,
	}
	
	for _, validType := range validTypes {
		if msgType == validType {
			return true
		}
	}
	return false
}
