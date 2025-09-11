package services

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"

	"notification-service/internal/config"
	"notification-service/internal/models"
)

// WebSocketService представляет сервис для WebSocket соединения с API Gateway
type WebSocketService struct {
	config          *config.Config
	logger          *logrus.Logger
	conn            *websocket.Conn
	connected       bool
	reconnectCount  int
	mu              sync.RWMutex
	stopCh          chan struct{}
	messageQueue    chan *models.WebSocketNotification
	connectionStatus string
}

// NewWebSocketService создает новый WebSocket сервис
func NewWebSocketService(config *config.Config, logger *logrus.Logger) *WebSocketService {
	return &WebSocketService{
		config:          config,
		logger:          logger,
		connected:       false,
		stopCh:          make(chan struct{}),
		messageQueue:    make(chan *models.WebSocketNotification, 1000),
		connectionStatus: models.StatusUnknown,
	}
}

// Start запускает WebSocket сервис
func (s *WebSocketService) Start() {
	s.logger.Info("Starting WebSocket service")
	
	// Запускаем горутину для обработки сообщений
	go s.messageProcessor()
	
	// Запускаем горутину для подключения
	go s.connectionManager()
}

// Stop останавливает WebSocket сервис
func (s *WebSocketService) Stop() {
	s.logger.Info("Stopping WebSocket service")
	
	close(s.stopCh)
	
	s.mu.Lock()
	if s.conn != nil {
		s.conn.Close()
		s.conn = nil
	}
	s.connected = false
	s.connectionStatus = models.StatusUnknown
	s.mu.Unlock()
}

// SendNotification отправляет уведомление через WebSocket
func (s *WebSocketService) SendNotification(notification *models.WebSocketNotification) error {
	select {
	case s.messageQueue <- notification:
		return nil
	default:
		s.logger.Warn("WebSocket message queue is full, dropping notification")
		return fmt.Errorf("message queue is full")
	}
}

// IsConnected проверяет, подключен ли WebSocket
func (s *WebSocketService) IsConnected() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.connected
}

// GetStatus возвращает статус соединения
func (s *WebSocketService) GetStatus() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.connectionStatus
}

// GetStats возвращает статистику WebSocket сервиса
func (s *WebSocketService) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	return map[string]interface{}{
		"connected":        s.connected,
		"reconnect_count":  s.reconnectCount,
		"connection_status": s.connectionStatus,
		"queue_size":       len(s.messageQueue),
		"queue_capacity":   cap(s.messageQueue),
	}
}

// connectionManager управляет WebSocket соединением
func (s *WebSocketService) connectionManager() {
	for {
		select {
		case <-s.stopCh:
			return
		default:
			if !s.IsConnected() {
				s.connect()
			}
			time.Sleep(time.Second)
		}
	}
}

// connect устанавливает WebSocket соединение
func (s *WebSocketService) connect() {
	if !s.config.WebSocket.ReconnectEnabled && s.reconnectCount > 0 {
		return
	}
	
	if s.reconnectCount >= s.config.WebSocket.MaxReconnectAttempts {
		s.logger.Error("Maximum reconnect attempts reached, stopping reconnection")
		s.setConnectionStatus(models.StatusUnhealthy)
		return
	}

	s.logger.WithField("attempt", s.reconnectCount+1).Info("Attempting to connect to API Gateway WebSocket")
	
	// Создаем HTTP заголовки
	headers := http.Header{}
	headers.Set("User-Agent", "notification-service/1.0")
	headers.Set("X-Service-Name", "notification-service")
	
	// Создаем диалер с таймаутом
	dialer := websocket.Dialer{
		HandshakeTimeout: s.config.WebSocket.ConnectTimeout,
	}
	
	// Подключаемся к API Gateway
	conn, _, err := dialer.Dial(s.config.WebSocket.GatewayURL, headers)
	if err != nil {
		s.logger.WithError(err).Error("Failed to connect to API Gateway WebSocket")
		s.setConnectionStatus(models.StatusUnhealthy)
		s.reconnectCount++
		
		if s.config.WebSocket.ReconnectEnabled {
			time.Sleep(s.config.WebSocket.ReconnectInterval)
		}
		return
	}
	
	s.mu.Lock()
	s.conn = conn
	s.connected = true
	s.connectionStatus = models.StatusHealthy
	s.reconnectCount = 0
	s.mu.Unlock()
	
	s.logger.Info("Successfully connected to API Gateway WebSocket")
	
	// Настраиваем таймауты
	conn.SetReadDeadline(time.Now().Add(s.config.WebSocket.ReadTimeout))
	conn.SetWriteDeadline(time.Now().Add(s.config.WebSocket.WriteTimeout))
	
	// Запускаем горутины для чтения и записи
	go s.readPump()
	go s.writePump()
	
	// Запускаем ping
	go s.pingRoutine()
}

// readPump читает сообщения от API Gateway
func (s *WebSocketService) readPump() {
	defer s.disconnect()
	
	s.mu.RLock()
	conn := s.conn
	s.mu.RUnlock()
	
	if conn == nil {
		return
	}
	
	conn.SetReadDeadline(time.Now().Add(s.config.WebSocket.ReadTimeout))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(s.config.WebSocket.ReadTimeout))
		return nil
	})
	
	for {
		var message map[string]interface{}
		if err := conn.ReadJSON(&message); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				s.logger.WithError(err).Error("WebSocket read error")
			}
			break
		}
		
		s.handleIncomingMessage(message)
	}
}

// writePump отправляет сообщения в API Gateway
func (s *WebSocketService) writePump() {
	defer s.disconnect()
	
	s.mu.RLock()
	conn := s.conn
	s.mu.RUnlock()
	
	if conn == nil {
		return
	}
	
	for {
		select {
		case <-s.stopCh:
			conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		default:
			// Этот pump в основном слушает ping от pingRoutine
			// Основные сообщения отправляются через messageProcessor
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// pingRoutine отправляет ping сообщения
func (s *WebSocketService) pingRoutine() {
	ticker := time.NewTicker(s.config.WebSocket.PingInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.mu.RLock()
			conn := s.conn
			connected := s.connected
			s.mu.RUnlock()
			
			if !connected || conn == nil {
				return
			}
			
			conn.SetWriteDeadline(time.Now().Add(s.config.WebSocket.WriteTimeout))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				s.logger.WithError(err).Error("Failed to send ping")
				return
			}
		}
	}
}

// messageProcessor обрабатывает очередь сообщений
func (s *WebSocketService) messageProcessor() {
	for {
		select {
		case <-s.stopCh:
			return
		case notification := <-s.messageQueue:
			s.sendMessage(notification)
		}
	}
}

// sendMessage отправляет сообщение через WebSocket
func (s *WebSocketService) sendMessage(notification *models.WebSocketNotification) {
	s.mu.RLock()
	conn := s.conn
	connected := s.connected
	s.mu.RUnlock()
	
	if !connected || conn == nil {
		s.logger.Debug("WebSocket not connected, message will be queued")
		return
	}
	
	// Создаем сообщение для API Gateway
	message := map[string]interface{}{
		"type": "notification_broadcast",
		"data": map[string]interface{}{
			"notification_type": notification.Type,
			"payload":          notification.Payload,
			"user_id":          notification.UserID,
			"timestamp":        time.Now(),
		},
		"user_id":   notification.UserID,
		"timestamp": time.Now(),
	}
	
	conn.SetWriteDeadline(time.Now().Add(s.config.WebSocket.WriteTimeout))
	if err := conn.WriteJSON(message); err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"user_id": notification.UserID,
			"type":    notification.Type,
		}).Error("Failed to send WebSocket message")
		
		// При ошибке отключаемся для переподключения
		s.disconnect()
		return
	}
	
	s.logger.WithFields(logrus.Fields{
		"user_id": notification.UserID,
		"type":    notification.Type,
	}).Debug("WebSocket message sent successfully")
}

// handleIncomingMessage обрабатывает входящие сообщения от API Gateway
func (s *WebSocketService) handleIncomingMessage(message map[string]interface{}) {
	messageType, ok := message["type"].(string)
	if !ok {
		s.logger.Warn("Received message without type")
		return
	}
	
	s.logger.WithField("type", messageType).Debug("Received WebSocket message")
	
	switch messageType {
	case "pong":
		// Pong обрабатывается автоматически
		
	case "connection_ack":
		s.logger.Info("WebSocket connection acknowledged by API Gateway")
		
	case "error":
		if errorMsg, ok := message["error"].(string); ok {
			s.logger.WithField("error", errorMsg).Error("Received error from API Gateway")
		}
		
	default:
		s.logger.WithField("type", messageType).Debug("Unknown message type from API Gateway")
	}
}

// disconnect отключается от WebSocket
func (s *WebSocketService) disconnect() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.conn != nil {
		s.conn.Close()
		s.conn = nil
	}
	
	if s.connected {
		s.connected = false
		s.connectionStatus = models.StatusUnhealthy
		s.logger.Warn("WebSocket disconnected from API Gateway")
		
		// Увеличиваем счетчик переподключений только при неожиданном отключении
		s.reconnectCount++
	}
}

// setConnectionStatus устанавливает статус соединения
func (s *WebSocketService) setConnectionStatus(status string) {
	s.mu.Lock()
	s.connectionStatus = status
	s.mu.Unlock()
}

// SendSystemBroadcast отправляет системное сообщение всем пользователям
func (s *WebSocketService) SendSystemBroadcast(messageType, title, message string, data map[string]interface{}) error {
	notification := &models.WebSocketNotification{
		Type: "system_broadcast",
		Payload: map[string]interface{}{
			"message_type": messageType,
			"title":        title,
			"message":      message,
			"data":         data,
			"timestamp":    time.Now(),
		},
	}
	
	return s.SendNotification(notification)
}

// SendUserNotification отправляет уведомление конкретному пользователю
func (s *WebSocketService) SendUserNotification(userID int, notificationType, title, message string, data map[string]interface{}) error {
	notification := &models.WebSocketNotification{
		Type:   notificationType,
		UserID: userID,
		Payload: map[string]interface{}{
			"title":     title,
			"message":   message,
			"data":      data,
			"timestamp": time.Now(),
		},
	}
	
	return s.SendNotification(notification)
}

// TestConnection тестирует WebSocket соединение
func (s *WebSocketService) TestConnection() error {
	if !s.IsConnected() {
		return fmt.Errorf("WebSocket is not connected")
	}
	
	// Отправляем тестовое сообщение
	testNotification := &models.WebSocketNotification{
		Type: "test",
		Payload: map[string]interface{}{
			"message":   "WebSocket connection test",
			"timestamp": time.Now(),
		},
	}
	
	return s.SendNotification(testNotification)
}
