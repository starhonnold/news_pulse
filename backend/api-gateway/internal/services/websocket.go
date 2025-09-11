package services

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"

	"api-gateway/internal/config"
	"api-gateway/internal/models"
)

// WebSocketService представляет сервис для управления WebSocket соединениями
type WebSocketService struct {
	config      *config.Config
	logger      *logrus.Logger
	upgrader    websocket.Upgrader
	connections map[string]*WebSocketClient
	userConns   map[int][]*WebSocketClient // соединения по пользователям
	mu          sync.RWMutex
	hub         chan *WebSocketMessage
	register    chan *WebSocketClient
	unregister  chan *WebSocketClient
}

// WebSocketClient представляет WebSocket клиента
type WebSocketClient struct {
	ID       string
	UserID   int
	Username string
	Conn     *websocket.Conn
	Send     chan *models.WebSocketMessage
	Service  *WebSocketService
	LastPing time.Time
	Active   bool
}

// WebSocketMessage представляет внутреннее сообщение WebSocket
type WebSocketMessage struct {
	Client  *WebSocketClient
	Message *models.WebSocketMessage
}

// NewWebSocketService создает новый WebSocket сервис
func NewWebSocketService(config *config.Config, logger *logrus.Logger) *WebSocketService {
	upgrader := websocket.Upgrader{
		ReadBufferSize:   config.WebSocket.ReadBufferSize,
		WriteBufferSize:  config.WebSocket.WriteBufferSize,
		HandshakeTimeout: config.WebSocket.HandshakeTimeout,
		CheckOrigin: func(r *http.Request) bool {
			// В production здесь должна быть более строгая проверка origin
			return true
		},
	}

	service := &WebSocketService{
		config:      config,
		logger:      logger,
		upgrader:    upgrader,
		connections: make(map[string]*WebSocketClient),
		userConns:   make(map[int][]*WebSocketClient),
		hub:         make(chan *WebSocketMessage, 1000),
		register:    make(chan *WebSocketClient, 100),
		unregister:  make(chan *WebSocketClient, 100),
	}

	// Запускаем hub в отдельной горутине
	go service.runHub()

	return service
}

// HandleWebSocket обрабатывает WebSocket соединения
func (s *WebSocketService) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	if !s.config.WebSocket.Enabled {
		http.Error(w, "WebSocket is disabled", http.StatusServiceUnavailable)
		return
	}

	// Проверяем лимит соединений
	if s.getTotalConnections() >= s.config.WebSocket.MaxConnections {
		http.Error(w, "Maximum connections exceeded", http.StatusServiceUnavailable)
		return
	}

	// Извлекаем информацию о пользователе (если есть)
	userID := s.getUserIDFromRequest(r)
	username := s.getUsernameFromRequest(r)

	// Проверяем лимит соединений на пользователя
	if userID > 0 && s.getUserConnectionCount(userID) >= s.config.WebSocket.MaxConnectionsPerUser {
		http.Error(w, "Maximum connections per user exceeded", http.StatusTooManyRequests)
		return
	}

	// Обновляем соединение до WebSocket
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.WithError(err).Error("Failed to upgrade connection to WebSocket")
		return
	}

	// Создаем клиента
	client := &WebSocketClient{
		ID:       s.generateClientID(),
		UserID:   userID,
		Username: username,
		Conn:     conn,
		Send:     make(chan *models.WebSocketMessage, 256),
		Service:  s,
		LastPing: time.Now(),
		Active:   true,
	}

	// Регистрируем клиента
	s.register <- client

	s.logger.WithFields(logrus.Fields{
		"client_id":   client.ID,
		"user_id":     client.UserID,
		"username":    client.Username,
		"remote_addr": r.RemoteAddr,
	}).Info("WebSocket client connected")

	// Запускаем горутины для чтения и записи
	go client.writePump()
	go client.readPump()
}

// runHub запускает центральный hub для управления соединениями
func (s *WebSocketService) runHub() {
	ticker := time.NewTicker(s.config.WebSocket.PingPeriod)
	defer ticker.Stop()

	for {
		select {
		case client := <-s.register:
			s.registerClient(client)

		case client := <-s.unregister:
			s.unregisterClient(client)

		case message := <-s.hub:
			s.handleMessage(message)

		case <-ticker.C:
			s.pingClients()
		}
	}
}

// registerClient регистрирует нового клиента
func (s *WebSocketService) registerClient(client *WebSocketClient) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.connections[client.ID] = client

	if client.UserID > 0 {
		s.userConns[client.UserID] = append(s.userConns[client.UserID], client)
	}

	s.logger.WithFields(logrus.Fields{
		"client_id":         client.ID,
		"user_id":           client.UserID,
		"total_connections": len(s.connections),
	}).Debug("WebSocket client registered")
}

// unregisterClient отменяет регистрацию клиента
func (s *WebSocketService) unregisterClient(client *WebSocketClient) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.connections[client.ID]; exists {
		delete(s.connections, client.ID)
		close(client.Send)

		// Удаляем из пользовательских соединений
		if client.UserID > 0 {
			userConns := s.userConns[client.UserID]
			for i, conn := range userConns {
				if conn.ID == client.ID {
					s.userConns[client.UserID] = append(userConns[:i], userConns[i+1:]...)
					break
				}
			}

			// Если у пользователя больше нет соединений, удаляем запись
			if len(s.userConns[client.UserID]) == 0 {
				delete(s.userConns, client.UserID)
			}
		}

		s.logger.WithFields(logrus.Fields{
			"client_id":         client.ID,
			"user_id":           client.UserID,
			"total_connections": len(s.connections),
		}).Debug("WebSocket client unregistered")
	}
}

// handleMessage обрабатывает входящее сообщение
func (s *WebSocketService) handleMessage(wsMessage *WebSocketMessage) {
	message := wsMessage.Message
	client := wsMessage.Client

	s.logger.WithFields(logrus.Fields{
		"client_id": client.ID,
		"user_id":   client.UserID,
		"type":      message.Type,
	}).Debug("Handling WebSocket message")

	switch message.Type {
	case models.WSMessageTypePing:
		s.handlePing(client)

	case models.WSMessageTypeNewsUpdate:
		// Обрабатываем обновления новостей
		s.handleNewsUpdate(message)

	case models.WSMessageTypePulseUpdate:
		// Обрабатываем обновления пульсов
		s.handlePulseUpdate(message)

	default:
		s.logger.WithFields(logrus.Fields{
			"client_id": client.ID,
			"type":      message.Type,
		}).Warn("Unknown WebSocket message type")
	}
}

// handlePing обрабатывает ping сообщения
func (s *WebSocketService) handlePing(client *WebSocketClient) {
	client.LastPing = time.Now()

	// Отправляем pong
	pongMessage := models.NewWebSocketMessage(models.WSMessageTypePong, map[string]interface{}{
		"timestamp": time.Now(),
	}, client.UserID)

	select {
	case client.Send <- pongMessage:
	default:
		// Канал заблокирован, закрываем соединение
		s.unregister <- client
	}
}

// handleNewsUpdate обрабатывает обновления новостей
func (s *WebSocketService) handleNewsUpdate(message *models.WebSocketMessage) {
	// Рассылаем обновление всем подключенным клиентам
	s.BroadcastToAll(message)
}

// handlePulseUpdate обрабатывает обновления пульсов
func (s *WebSocketService) handlePulseUpdate(message *models.WebSocketMessage) {
	// Рассылаем обновление конкретному пользователю
	if message.UserID > 0 {
		s.SendToUser(message.UserID, message)
	}
}

// pingClients отправляет ping всем клиентам
func (s *WebSocketService) pingClients() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()

	for _, client := range s.connections {
		// Проверяем, не истек ли таймаут
		if now.Sub(client.LastPing) > s.config.WebSocket.PongWait {
			s.logger.WithField("client_id", client.ID).Debug("Client ping timeout, closing connection")
			s.unregister <- client
			continue
		}

		// Отправляем ping
		pingMessage := models.NewWebSocketMessage(models.WSMessageTypePing, map[string]interface{}{
			"timestamp": now,
		}, client.UserID)

		select {
		case client.Send <- pingMessage:
		default:
			s.unregister <- client
		}
	}
}

// BroadcastToAll рассылает сообщение всем подключенным клиентам
func (s *WebSocketService) BroadcastToAll(message *models.WebSocketMessage) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, client := range s.connections {
		select {
		case client.Send <- message:
		default:
			s.unregister <- client
		}
	}

	s.logger.WithFields(logrus.Fields{
		"type":       message.Type,
		"recipients": len(s.connections),
	}).Debug("Broadcasted message to all clients")
}

// SendToUser отправляет сообщение конкретному пользователю
func (s *WebSocketService) SendToUser(userID int, message *models.WebSocketMessage) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	userConns, exists := s.userConns[userID]
	if !exists {
		s.logger.WithField("user_id", userID).Debug("No active connections for user")
		return
	}

	sent := 0
	for _, client := range userConns {
		select {
		case client.Send <- message:
			sent++
		default:
			s.unregister <- client
		}
	}

	s.logger.WithFields(logrus.Fields{
		"user_id": userID,
		"type":    message.Type,
		"sent":    sent,
		"total":   len(userConns),
	}).Debug("Sent message to user")
}

// GetStats возвращает статистику WebSocket сервиса
func (s *WebSocketService) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"enabled":           s.config.WebSocket.Enabled,
		"total_connections": len(s.connections),
		"unique_users":      len(s.userConns),
		"max_connections":   s.config.WebSocket.MaxConnections,
		"max_per_user":      s.config.WebSocket.MaxConnectionsPerUser,
	}
}

// Методы WebSocketClient

// readPump читает сообщения от клиента
func (c *WebSocketClient) readPump() {
	defer func() {
		c.Service.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(512)
	c.Conn.SetReadDeadline(time.Now().Add(c.Service.config.WebSocket.PongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(c.Service.config.WebSocket.PongWait))
		c.LastPing = time.Now()
		return nil
	})

	for {
		var message models.WebSocketMessage
		if err := c.Conn.ReadJSON(&message); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.Service.logger.WithError(err).Error("WebSocket read error")
			}
			break
		}

		// Добавляем информацию о пользователе
		message.UserID = c.UserID
		message.Timestamp = time.Now()

		// Отправляем сообщение в hub
		c.Service.hub <- &WebSocketMessage{
			Client:  c,
			Message: &message,
		}
	}
}

// writePump отправляет сообщения клиенту
func (c *WebSocketClient) writePump() {
	ticker := time.NewTicker(c.Service.config.WebSocket.PingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(c.Service.config.WebSocket.WriteWait))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteJSON(message); err != nil {
				c.Service.logger.WithError(err).Error("WebSocket write error")
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(c.Service.config.WebSocket.WriteWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Вспомогательные методы

// generateClientID генерирует уникальный ID для клиента
func (s *WebSocketService) generateClientID() string {
	return fmt.Sprintf("ws_%d_%d", time.Now().UnixNano(), len(s.connections))
}

// getTotalConnections возвращает общее количество соединений
func (s *WebSocketService) getTotalConnections() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.connections)
}

// getUserConnectionCount возвращает количество соединений пользователя
func (s *WebSocketService) getUserConnectionCount(userID int) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.userConns[userID])
}

// getUserIDFromRequest извлекает ID пользователя из запроса
func (s *WebSocketService) getUserIDFromRequest(r *http.Request) int {
	// В реальном приложении здесь должна быть проверка JWT токена из query параметра или cookie
	// Для демонстрации используем заголовок
	if userIDStr := r.Header.Get("X-User-ID"); userIDStr != "" {
		var userID int
		if _, err := fmt.Sscanf(userIDStr, "%d", &userID); err == nil {
			return userID
		}
	}
	return 0
}

// getUsernameFromRequest извлекает имя пользователя из запроса
func (s *WebSocketService) getUsernameFromRequest(r *http.Request) string {
	return r.Header.Get("X-Username")
}
