package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"notification-service/internal/models"
)

// NotificationRepository представляет репозиторий для уведомлений
type NotificationRepository struct {
	db     *sql.DB
	logger *logrus.Logger
}

// NewNotificationRepository создает новый репозиторий уведомлений
func NewNotificationRepository(db *sql.DB, logger *logrus.Logger) *NotificationRepository {
	return &NotificationRepository{
		db:     db,
		logger: logger,
	}
}

// Create создает новое уведомление
func (r *NotificationRepository) Create(notification *models.Notification) error {
	query := `
		INSERT INTO notifications (user_id, type, title, message, data, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`
	
	var dataJSON []byte
	if notification.Data != "" {
		dataJSON = []byte(notification.Data)
	}
	
	err := r.db.QueryRow(
		query,
		notification.UserID,
		notification.Type,
		notification.Title,
		notification.Message,
		dataJSON,
		notification.ExpiresAt,
	).Scan(&notification.ID, &notification.CreatedAt)
	
	if err != nil {
		r.logger.WithError(err).WithFields(logrus.Fields{
			"user_id": notification.UserID,
			"type":    notification.Type,
		}).Error("Failed to create notification")
		return fmt.Errorf("failed to create notification: %w", err)
	}
	
	r.logger.WithFields(logrus.Fields{
		"notification_id": notification.ID,
		"user_id":         notification.UserID,
		"type":            notification.Type,
	}).Debug("Notification created successfully")
	
	return nil
}

// GetByID получает уведомление по ID
func (r *NotificationRepository) GetByID(id int) (*models.Notification, error) {
	query := `
		SELECT id, user_id, type, title, message, data, is_read, created_at, read_at, expires_at
		FROM notifications
		WHERE id = $1`
	
	notification := &models.Notification{}
	var dataJSON []byte
	
	err := r.db.QueryRow(query, id).Scan(
		&notification.ID,
		&notification.UserID,
		&notification.Type,
		&notification.Title,
		&notification.Message,
		&dataJSON,
		&notification.IsRead,
		&notification.CreatedAt,
		&notification.ReadAt,
		&notification.ExpiresAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("notification not found")
		}
		r.logger.WithError(err).WithField("id", id).Error("Failed to get notification by ID")
		return nil, fmt.Errorf("failed to get notification: %w", err)
	}
	
	if dataJSON != nil {
		notification.Data = string(dataJSON)
	}
	
	return notification, nil
}

// GetByUserID получает уведомления пользователя с фильтрацией и пагинацией
func (r *NotificationRepository) GetByUserID(userID int, filter *models.NotificationFilter) ([]models.Notification, int, error) {
	// Строим запрос с фильтрами
	whereConditions := []string{"user_id = $1"}
	args := []interface{}{userID}
	argIndex := 2
	
	if filter.Type != "" {
		whereConditions = append(whereConditions, fmt.Sprintf("type = $%d", argIndex))
		args = append(args, filter.Type)
		argIndex++
	}
	
	if filter.IsRead != nil {
		whereConditions = append(whereConditions, fmt.Sprintf("is_read = $%d", argIndex))
		args = append(args, *filter.IsRead)
		argIndex++
	}
	
	if filter.DateFrom != nil {
		whereConditions = append(whereConditions, fmt.Sprintf("created_at >= $%d", argIndex))
		args = append(args, *filter.DateFrom)
		argIndex++
	}
	
	if filter.DateTo != nil {
		whereConditions = append(whereConditions, fmt.Sprintf("created_at <= $%d", argIndex))
		args = append(args, *filter.DateTo)
		argIndex++
	}
	
	whereClause := strings.Join(whereConditions, " AND ")
	
	// Получаем общее количество записей
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM notifications WHERE %s", whereClause)
	var total int
	err := r.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		r.logger.WithError(err).Error("Failed to count notifications")
		return nil, 0, fmt.Errorf("failed to count notifications: %w", err)
	}
	
	// Получаем уведомления с пагинацией
	offset := (filter.Page - 1) * filter.PageSize
	dataQuery := fmt.Sprintf(`
		SELECT id, user_id, type, title, message, data, is_read, created_at, read_at, expires_at
		FROM notifications
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`,
		whereClause, argIndex, argIndex+1)
	
	args = append(args, filter.PageSize, offset)
	
	rows, err := r.db.Query(dataQuery, args...)
	if err != nil {
		r.logger.WithError(err).WithField("user_id", userID).Error("Failed to get notifications")
		return nil, 0, fmt.Errorf("failed to get notifications: %w", err)
	}
	defer rows.Close()
	
	var notifications []models.Notification
	
	for rows.Next() {
		notification := models.Notification{}
		var dataJSON []byte
		
		err := rows.Scan(
			&notification.ID,
			&notification.UserID,
			&notification.Type,
			&notification.Title,
			&notification.Message,
			&dataJSON,
			&notification.IsRead,
			&notification.CreatedAt,
			&notification.ReadAt,
			&notification.ExpiresAt,
		)
		
		if err != nil {
			r.logger.WithError(err).Error("Failed to scan notification row")
			continue
		}
		
		if dataJSON != nil {
			notification.Data = string(dataJSON)
		}
		
		notifications = append(notifications, notification)
	}
	
	if err = rows.Err(); err != nil {
		r.logger.WithError(err).Error("Error iterating notification rows")
		return nil, 0, fmt.Errorf("failed to iterate notifications: %w", err)
	}
	
	return notifications, total, nil
}

// Update обновляет уведомление
func (r *NotificationRepository) Update(id int, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return fmt.Errorf("no updates provided")
	}
	
	setParts := make([]string, 0, len(updates))
	args := make([]interface{}, 0, len(updates)+1)
	argIndex := 1
	
	for field, value := range updates {
		setParts = append(setParts, fmt.Sprintf("%s = $%d", field, argIndex))
		args = append(args, value)
		argIndex++
	}
	
	// Добавляем updated_at если его нет
	if _, exists := updates["updated_at"]; !exists {
		setParts = append(setParts, fmt.Sprintf("updated_at = $%d", argIndex))
		args = append(args, time.Now())
		argIndex++
	}
	
	query := fmt.Sprintf(
		"UPDATE notifications SET %s WHERE id = $%d",
		strings.Join(setParts, ", "),
		argIndex,
	)
	args = append(args, id)
	
	result, err := r.db.Exec(query, args...)
	if err != nil {
		r.logger.WithError(err).WithField("id", id).Error("Failed to update notification")
		return fmt.Errorf("failed to update notification: %w", err)
	}
	
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("notification not found")
	}
	
	r.logger.WithField("notification_id", id).Debug("Notification updated successfully")
	return nil
}

// MarkAsRead помечает уведомление как прочитанное
func (r *NotificationRepository) MarkAsRead(id int) error {
	updates := map[string]interface{}{
		"is_read": true,
		"read_at": time.Now(),
	}
	
	return r.Update(id, updates)
}

// MarkAllAsRead помечает все уведомления пользователя как прочитанные
func (r *NotificationRepository) MarkAllAsRead(userID int) error {
	query := `
		UPDATE notifications 
		SET is_read = true, read_at = CURRENT_TIMESTAMP
		WHERE user_id = $1 AND is_read = false`
	
	result, err := r.db.Exec(query, userID)
	if err != nil {
		r.logger.WithError(err).WithField("user_id", userID).Error("Failed to mark all notifications as read")
		return fmt.Errorf("failed to mark all notifications as read: %w", err)
	}
	
	rowsAffected, _ := result.RowsAffected()
	r.logger.WithFields(logrus.Fields{
		"user_id":       userID,
		"updated_count": rowsAffected,
	}).Debug("Marked all notifications as read")
	
	return nil
}

// Delete удаляет уведомление
func (r *NotificationRepository) Delete(id int) error {
	query := "DELETE FROM notifications WHERE id = $1"
	
	result, err := r.db.Exec(query, id)
	if err != nil {
		r.logger.WithError(err).WithField("id", id).Error("Failed to delete notification")
		return fmt.Errorf("failed to delete notification: %w", err)
	}
	
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("notification not found")
	}
	
	r.logger.WithField("notification_id", id).Debug("Notification deleted successfully")
	return nil
}

// DeleteByUserID удаляет все уведомления пользователя
func (r *NotificationRepository) DeleteByUserID(userID int) error {
	query := "DELETE FROM notifications WHERE user_id = $1"
	
	result, err := r.db.Exec(query, userID)
	if err != nil {
		r.logger.WithError(err).WithField("user_id", userID).Error("Failed to delete user notifications")
		return fmt.Errorf("failed to delete user notifications: %w", err)
	}
	
	rowsAffected, _ := result.RowsAffected()
	r.logger.WithFields(logrus.Fields{
		"user_id":       userID,
		"deleted_count": rowsAffected,
	}).Debug("User notifications deleted successfully")
	
	return nil
}

// GetStats возвращает статистику уведомлений пользователя
func (r *NotificationRepository) GetStats(userID int) (*models.NotificationStats, error) {
	stats := &models.NotificationStats{
		NotificationsByType: make(map[string]int),
		RecentActivity:      make([]models.NotificationActivity, 0),
	}
	
	// Общее количество уведомлений
	err := r.db.QueryRow(
		"SELECT COUNT(*) FROM notifications WHERE user_id = $1",
		userID,
	).Scan(&stats.TotalNotifications)
	if err != nil {
		return nil, fmt.Errorf("failed to get total notifications: %w", err)
	}
	
	// Количество непрочитанных уведомлений
	err = r.db.QueryRow(
		"SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = false",
		userID,
	).Scan(&stats.UnreadNotifications)
	if err != nil {
		return nil, fmt.Errorf("failed to get unread notifications: %w", err)
	}
	
	// Статистика по типам
	typeQuery := `
		SELECT type, COUNT(*) 
		FROM notifications 
		WHERE user_id = $1 
		GROUP BY type`
	
	rows, err := r.db.Query(typeQuery, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get notifications by type: %w", err)
	}
	defer rows.Close()
	
	for rows.Next() {
		var notificationType string
		var count int
		
		if err := rows.Scan(&notificationType, &count); err != nil {
			r.logger.WithError(err).Error("Failed to scan notification type stats")
			continue
		}
		
		stats.NotificationsByType[notificationType] = count
	}
	
	// Активность за последние 7 дней
	activityQuery := `
		SELECT DATE(created_at) as date, COUNT(*) as count
		FROM notifications 
		WHERE user_id = $1 AND created_at >= CURRENT_DATE - INTERVAL '7 days'
		GROUP BY DATE(created_at)
		ORDER BY date DESC`
	
	rows, err = r.db.Query(activityQuery, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent activity: %w", err)
	}
	defer rows.Close()
	
	for rows.Next() {
		var activity models.NotificationActivity
		
		if err := rows.Scan(&activity.Date, &activity.Count); err != nil {
			r.logger.WithError(err).Error("Failed to scan notification activity")
			continue
		}
		
		stats.RecentActivity = append(stats.RecentActivity, activity)
	}
	
	return stats, nil
}

// CreateBatch создает несколько уведомлений за один запрос
func (r *NotificationRepository) CreateBatch(notifications []models.Notification) error {
	if len(notifications) == 0 {
		return nil
	}
	
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	query := `
		INSERT INTO notifications (user_id, type, title, message, data, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`
	
	stmt, err := tx.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()
	
	for i := range notifications {
		notification := &notifications[i]
		
		var dataJSON []byte
		if notification.Data != "" {
			dataJSON = []byte(notification.Data)
		}
		
		err := stmt.QueryRow(
			notification.UserID,
			notification.Type,
			notification.Title,
			notification.Message,
			dataJSON,
			notification.ExpiresAt,
		).Scan(&notification.ID, &notification.CreatedAt)
		
		if err != nil {
			r.logger.WithError(err).WithFields(logrus.Fields{
				"user_id": notification.UserID,
				"type":    notification.Type,
			}).Error("Failed to create notification in batch")
			return fmt.Errorf("failed to create notification in batch: %w", err)
		}
	}
	
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	r.logger.WithField("count", len(notifications)).Debug("Batch notifications created successfully")
	return nil
}

// GetExpiredNotifications получает истекшие уведомления
func (r *NotificationRepository) GetExpiredNotifications(limit int) ([]models.Notification, error) {
	query := `
		SELECT id, user_id, type, title, message, data, is_read, created_at, read_at, expires_at
		FROM notifications
		WHERE expires_at IS NOT NULL AND expires_at < CURRENT_TIMESTAMP
		ORDER BY expires_at ASC
		LIMIT $1`
	
	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get expired notifications: %w", err)
	}
	defer rows.Close()
	
	var notifications []models.Notification
	
	for rows.Next() {
		notification := models.Notification{}
		var dataJSON []byte
		
		err := rows.Scan(
			&notification.ID,
			&notification.UserID,
			&notification.Type,
			&notification.Title,
			&notification.Message,
			&dataJSON,
			&notification.IsRead,
			&notification.CreatedAt,
			&notification.ReadAt,
			&notification.ExpiresAt,
		)
		
		if err != nil {
			r.logger.WithError(err).Error("Failed to scan expired notification")
			continue
		}
		
		if dataJSON != nil {
			notification.Data = string(dataJSON)
		}
		
		notifications = append(notifications, notification)
	}
	
	return notifications, nil
}

// GetUnreadCount возвращает количество непрочитанных уведомлений пользователя
func (r *NotificationRepository) GetUnreadCount(userID int) (int, error) {
	var count int
	err := r.db.QueryRow(
		"SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = false",
		userID,
	).Scan(&count)
	
	if err != nil {
		r.logger.WithError(err).WithField("user_id", userID).Error("Failed to get unread count")
		return 0, fmt.Errorf("failed to get unread count: %w", err)
	}
	
	return count, nil
}
