package services

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/sirupsen/logrus"

	"notification-service/internal/config"
	"notification-service/internal/models"
	"notification-service/internal/repository"
)

// NotificationService представляет сервис для управления уведомлениями
type NotificationService struct {
	config                *config.Config
	logger                *logrus.Logger
	notificationRepo      *repository.NotificationRepository
	websocketService      *WebSocketService
	eventProcessor        *EventProcessor
	templates             map[string]*template.Template
}

// NewNotificationService создает новый сервис уведомлений
func NewNotificationService(
	config *config.Config,
	logger *logrus.Logger,
	notificationRepo *repository.NotificationRepository,
	websocketService *WebSocketService,
) *NotificationService {
	service := &NotificationService{
		config:           config,
		logger:           logger,
		notificationRepo: notificationRepo,
		websocketService: websocketService,
		templates:        make(map[string]*template.Template),
	}

	// Загружаем шаблоны уведомлений
	service.loadTemplates()

	// Создаем процессор событий
	service.eventProcessor = NewEventProcessor(config, logger, service)

	return service
}

// CreateNotification создает новое уведомление
func (s *NotificationService) CreateNotification(req *models.CreateNotificationRequest) (*models.Notification, error) {
	// Валидируем запрос
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Проверяем валидность типа уведомления
	if !s.config.IsNotificationTypeValid(req.Type) {
		return nil, models.NewAPIError(
			models.ErrorCodeValidation,
			fmt.Sprintf("Invalid notification type: %s", req.Type),
		)
	}

	// Проверяем лимит уведомлений на пользователя
	if err := s.checkUserNotificationLimit(req.UserID); err != nil {
		return nil, err
	}

	// Создаем уведомление
	notification := &models.Notification{
		UserID:    req.UserID,
		Type:      req.Type,
		Title:     req.Title,
		Message:   req.Message,
		IsRead:    false,
		ExpiresAt: req.ExpiresAt,
	}

	// Сериализуем дополнительные данные
	if req.Data != nil {
		dataJSON, err := json.Marshal(req.Data)
		if err != nil {
			s.logger.WithError(err).Error("Failed to marshal notification data")
			return nil, models.NewAPIError(models.ErrorCodeInternalError, "Failed to process notification data")
		}
		notification.Data = string(dataJSON)
	}

	// Сохраняем в базе данных
	if err := s.notificationRepo.Create(notification); err != nil {
		s.logger.WithError(err).WithField("user_id", req.UserID).Error("Failed to create notification")
		return nil, models.NewAPIError(models.ErrorCodeDatabaseError, "Failed to create notification")
	}

	// Отправляем через WebSocket
	s.sendWebSocketNotification(notification)

	s.logger.WithFields(logrus.Fields{
		"notification_id": notification.ID,
		"user_id":         notification.UserID,
		"type":            notification.Type,
	}).Info("Notification created successfully")

	return notification, nil
}

// GetNotification получает уведомление по ID
func (s *NotificationService) GetNotification(id int) (*models.Notification, error) {
	notification, err := s.notificationRepo.GetByID(id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, models.NewAPIError(models.ErrorCodeNotFound, "Notification not found")
		}
		return nil, models.NewAPIError(models.ErrorCodeDatabaseError, "Failed to get notification")
	}

	return notification, nil
}

// GetUserNotifications получает уведомления пользователя
func (s *NotificationService) GetUserNotifications(userID int, filter *models.NotificationFilter) (*models.PaginatedNotifications, error) {
	// Валидируем фильтр
	if err := filter.Validate(); err != nil {
		return nil, err
	}

	// Получаем уведомления из базы данных
	notifications, total, err := s.notificationRepo.GetByUserID(userID, filter)
	if err != nil {
		s.logger.WithError(err).WithField("user_id", userID).Error("Failed to get user notifications")
		return nil, models.NewAPIError(models.ErrorCodeDatabaseError, "Failed to get notifications")
	}

	// Вычисляем количество страниц
	totalPages := (total + filter.PageSize - 1) / filter.PageSize

	return &models.PaginatedNotifications{
		Notifications: notifications,
		Total:         total,
		Page:          filter.Page,
		PageSize:      filter.PageSize,
		TotalPages:    totalPages,
	}, nil
}

// MarkNotificationAsRead помечает уведомление как прочитанное
func (s *NotificationService) MarkNotificationAsRead(id int) error {
	if err := s.notificationRepo.MarkAsRead(id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return models.NewAPIError(models.ErrorCodeNotFound, "Notification not found")
		}
		s.logger.WithError(err).WithField("id", id).Error("Failed to mark notification as read")
		return models.NewAPIError(models.ErrorCodeDatabaseError, "Failed to update notification")
	}

	s.logger.WithField("notification_id", id).Debug("Notification marked as read")
	return nil
}

// MarkAllNotificationsAsRead помечает все уведомления пользователя как прочитанные
func (s *NotificationService) MarkAllNotificationsAsRead(userID int) error {
	if err := s.notificationRepo.MarkAllAsRead(userID); err != nil {
		s.logger.WithError(err).WithField("user_id", userID).Error("Failed to mark all notifications as read")
		return models.NewAPIError(models.ErrorCodeDatabaseError, "Failed to update notifications")
	}

	s.logger.WithField("user_id", userID).Debug("All notifications marked as read")
	return nil
}

// DeleteNotification удаляет уведомление
func (s *NotificationService) DeleteNotification(id int) error {
	if err := s.notificationRepo.Delete(id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return models.NewAPIError(models.ErrorCodeNotFound, "Notification not found")
		}
		s.logger.WithError(err).WithField("id", id).Error("Failed to delete notification")
		return models.NewAPIError(models.ErrorCodeDatabaseError, "Failed to delete notification")
	}

	s.logger.WithField("notification_id", id).Debug("Notification deleted")
	return nil
}

// GetNotificationStats возвращает статистику уведомлений пользователя
func (s *NotificationService) GetNotificationStats(userID int) (*models.NotificationStats, error) {
	stats, err := s.notificationRepo.GetStats(userID)
	if err != nil {
		s.logger.WithError(err).WithField("user_id", userID).Error("Failed to get notification stats")
		return nil, models.NewAPIError(models.ErrorCodeDatabaseError, "Failed to get statistics")
	}

	return stats, nil
}

// ProcessEvent обрабатывает событие для создания уведомлений
func (s *NotificationService) ProcessEvent(event *models.NotificationEvent) error {
	s.logger.WithFields(logrus.Fields{
		"event_id":   event.ID,
		"event_type": event.Type,
		"user_id":    event.UserID,
	}).Debug("Processing notification event")

	// Если указан конкретный пользователь
	if event.UserID > 0 {
		return s.createNotificationFromEvent(event, event.UserID)
	}

	// Если указан список пользователей
	if len(event.UserIDs) > 0 {
		var notifications []models.Notification
		
		for _, userID := range event.UserIDs {
			notification := s.buildNotificationFromEvent(event, userID)
			notifications = append(notifications, *notification)
		}

		// Создаем уведомления батчем
		if err := s.notificationRepo.CreateBatch(notifications); err != nil {
			s.logger.WithError(err).Error("Failed to create batch notifications")
			return err
		}

		// Отправляем WebSocket уведомления
		for _, notification := range notifications {
			s.sendWebSocketNotification(&notification)
		}

		s.logger.WithField("count", len(notifications)).Info("Batch notifications created")
		return nil
	}

	return models.NewAPIError(models.ErrorCodeValidation, "No target users specified in event")
}

// CreateNewsAlert создает уведомление о новой важной новости
func (s *NotificationService) CreateNewsAlert(userID int, data *models.NewsAlertData) error {
	// Применяем шаблон
	title, message, err := s.renderTemplate("news_alert", map[string]interface{}{
		"Title":   data.Title,
		"Summary": data.Summary,
	})
	if err != nil {
		return err
	}

	// Создаем событие
	event := &models.NotificationEvent{
		ID:      fmt.Sprintf("news_alert_%d_%d", data.NewsID, time.Now().Unix()),
		Type:    models.NotificationTypeNewsAlert,
		UserID:  userID,
		Title:   title,
		Message: message,
		Data: map[string]interface{}{
			"news_id":      data.NewsID,
			"news_url":     data.URL,
			"source_name":  data.SourceName,
			"category":     data.Category,
			"published_at": data.PublishedAt,
		},
		CreatedAt: time.Now(),
	}

	return s.ProcessEvent(event)
}

// CreatePulseUpdate создает уведомление об обновлении пульса
func (s *NotificationService) CreatePulseUpdate(userID int, data *models.PulseUpdateData) error {
	// Применяем шаблон
	title, message, err := s.renderTemplate("pulse_update", map[string]interface{}{
		"PulseName":  data.PulseName,
		"NewsCount":  data.NewsCount,
	})
	if err != nil {
		return err
	}

	// Создаем событие
	event := &models.NotificationEvent{
		ID:      fmt.Sprintf("pulse_update_%d_%d", data.PulseID, time.Now().Unix()),
		Type:    models.NotificationTypePulseUpdate,
		UserID:  userID,
		Title:   title,
		Message: message,
		Data: map[string]interface{}{
			"pulse_id":    data.PulseID,
			"pulse_name":  data.PulseName,
			"news_count":  data.NewsCount,
			"update_type": data.UpdateType,
		},
		CreatedAt: time.Now(),
	}

	return s.ProcessEvent(event)
}

// CreateSystemMessage создает системное уведомление
func (s *NotificationService) CreateSystemMessage(userIDs []int, message string, data *models.SystemMessageData) error {
	// Применяем шаблон
	title, renderedMessage, err := s.renderTemplate("system_message", map[string]interface{}{
		"Message": message,
	})
	if err != nil {
		return err
	}

	// Создаем событие
	event := &models.NotificationEvent{
		ID:      fmt.Sprintf("system_message_%d", time.Now().Unix()),
		Type:    models.NotificationTypeSystemMessage,
		UserIDs: userIDs,
		Title:   title,
		Message: renderedMessage,
		Data: map[string]interface{}{
			"message_type": data.MessageType,
			"priority":     data.Priority,
			"action_url":   data.ActionURL,
			"metadata":     data.Metadata,
		},
		Priority:  data.Priority,
		CreatedAt: time.Now(),
	}

	return s.ProcessEvent(event)
}

// GetUnreadCount возвращает количество непрочитанных уведомлений пользователя
func (s *NotificationService) GetUnreadCount(userID int) (int, error) {
	count, err := s.notificationRepo.GetUnreadCount(userID)
	if err != nil {
		return 0, models.NewAPIError(models.ErrorCodeDatabaseError, "Failed to get unread count")
	}

	return count, nil
}

// Приватные методы

// checkUserNotificationLimit проверяет лимит уведомлений на пользователя
func (s *NotificationService) checkUserNotificationLimit(userID int) error {
	// Получаем текущее количество уведомлений пользователя
	filter := &models.NotificationFilter{
		UserID:   userID,
		Page:     1,
		PageSize: 1,
	}
	
	_, total, err := s.notificationRepo.GetByUserID(userID, filter)
	if err != nil {
		return models.NewAPIError(models.ErrorCodeDatabaseError, "Failed to check notification limit")
	}

	if total >= s.config.Notifications.MaxNotificationsPerUser {
		return models.NewAPIError(
			models.ErrorCodeValidation,
			fmt.Sprintf("User notification limit exceeded (%d)", s.config.Notifications.MaxNotificationsPerUser),
		)
	}

	return nil
}

// createNotificationFromEvent создает уведомление из события
func (s *NotificationService) createNotificationFromEvent(event *models.NotificationEvent, userID int) error {
	notification := s.buildNotificationFromEvent(event, userID)
	
	if err := s.notificationRepo.Create(notification); err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"event_id": event.ID,
			"user_id":  userID,
		}).Error("Failed to create notification from event")
		return err
	}

	// Отправляем через WebSocket
	s.sendWebSocketNotification(notification)

	return nil
}

// buildNotificationFromEvent строит уведомление из события
func (s *NotificationService) buildNotificationFromEvent(event *models.NotificationEvent, userID int) *models.Notification {
	notification := &models.Notification{
		UserID:    userID,
		Type:      event.Type,
		Title:     event.Title,
		Message:   event.Message,
		IsRead:    false,
		ExpiresAt: event.ExpiresAt,
	}

	// Сериализуем данные
	if event.Data != nil {
		if dataJSON, err := json.Marshal(event.Data); err == nil {
			notification.Data = string(dataJSON)
		}
	}

	return notification
}

// sendWebSocketNotification отправляет уведомление через WebSocket
func (s *NotificationService) sendWebSocketNotification(notification *models.Notification) {
	wsNotification := models.NewWebSocketNotification(
		"notification_created",
		notification,
		notification.UserID,
	)

	if err := s.websocketService.SendNotification(wsNotification); err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"notification_id": notification.ID,
			"user_id":         notification.UserID,
		}).Error("Failed to send WebSocket notification")
	}
}

// loadTemplates загружает шаблоны уведомлений
func (s *NotificationService) loadTemplates() {
	templates := map[string]config.TemplateConfig{
		"news_alert":     s.config.Templates.NewsAlert,
		"pulse_update":   s.config.Templates.PulseUpdate,
		"system_message": s.config.Templates.SystemMessage,
	}

	for name, templateConfig := range templates {
		// Создаем составной шаблон
		compositeTemplate := template.New(name)
		
		// Добавляем шаблон для заголовка
		_, err := compositeTemplate.New("title").Parse(templateConfig.Title)
		if err != nil {
			s.logger.WithError(err).WithField("template", name).Error("Failed to parse title template")
			continue
		}

		// Добавляем шаблон для сообщения
		_, err = compositeTemplate.New("body").Parse(templateConfig.Body)
		if err != nil {
			s.logger.WithError(err).WithField("template", name).Error("Failed to parse body template")
			continue
		}

		s.templates[name] = compositeTemplate
	}

	s.logger.WithField("count", len(s.templates)).Info("Notification templates loaded")
}

// renderTemplate применяет шаблон к данным
func (s *NotificationService) renderTemplate(templateName string, data map[string]interface{}) (string, string, error) {
	tmpl, exists := s.templates[templateName]
	if !exists {
		return "", "", fmt.Errorf("template %s not found", templateName)
	}

	// Рендерим заголовок
	var titleBuf strings.Builder
	if err := tmpl.Lookup("title").Execute(&titleBuf, data); err != nil {
		return "", "", fmt.Errorf("failed to render title template: %w", err)
	}

	// Рендерим сообщение
	var bodyBuf strings.Builder
	if err := tmpl.Lookup("body").Execute(&bodyBuf, data); err != nil {
		return "", "", fmt.Errorf("failed to render body template: %w", err)
	}

	return titleBuf.String(), bodyBuf.String(), nil
}

// Start запускает сервис уведомлений
func (s *NotificationService) Start() {
	s.logger.Info("Starting notification service")
	
	// Запускаем процессор событий
	s.eventProcessor.Start()
}

// Stop останавливает сервис уведомлений
func (s *NotificationService) Stop() {
	s.logger.Info("Stopping notification service")
	
	// Останавливаем процессор событий
	if s.eventProcessor != nil {
		s.eventProcessor.Stop()
	}
}
