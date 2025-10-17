package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// NewsSource представляет источник новостей
type NewsSource struct {
	ID                   int        `json:"id" db:"id"`
	Name                 string     `json:"name" db:"name"`
	Domain               string     `json:"domain" db:"domain"`
	RSSURL               string     `json:"rss_url" db:"rss_url"`
	WebsiteURL           string     `json:"website_url" db:"website_url"`
	CountryID            int        `json:"country_id" db:"country_id"`
	Language             string     `json:"language" db:"language"`
	Description          string     `json:"description" db:"description"`
	LogoURL              string     `json:"logo_url" db:"logo_url"`
	IsActive             bool       `json:"is_active" db:"is_active"`
	LastParsedAt         *time.Time `json:"last_parsed_at" db:"last_parsed_at"`
	ParseIntervalMinutes int        `json:"parse_interval_minutes" db:"parse_interval_minutes"`
	CreatedAt            time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at" db:"updated_at"`
}

// News представляет новость
type News struct {
	ID             int       `json:"id" db:"id"`
	Title          string    `json:"title" db:"title"`
	Description    string    `json:"description" db:"description"`
	Content        string    `json:"content" db:"content"`
	URL            string    `json:"url" db:"url"`
	ImageURL       string    `json:"image_url" db:"image_url"`
	Author         string    `json:"author" db:"author"`
	SourceID       int       `json:"source_id" db:"source_id"`
	CategoryID     *int      `json:"category_id" db:"category_id"`
	PublishedAt    time.Time `json:"published_at" db:"published_at"`
	ParsedAt       time.Time `json:"parsed_at" db:"parsed_at"`
	RelevanceScore float64   `json:"relevance_score" db:"relevance_score"`
	ViewCount      int       `json:"view_count" db:"view_count"`
	IsActive       bool      `json:"is_active" db:"is_active"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`

	// Связанные данные (не хранятся в БД)
	Source   *NewsSource `json:"source,omitempty" db:"-"`
	Category *Category   `json:"category,omitempty" db:"-"`
	Tags     []Tag       `json:"tags,omitempty" db:"-"`
}

// Category представляет категорию новости
type Category struct {
	ID          int       `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Slug        string    `json:"slug" db:"slug"`
	Color       string    `json:"color" db:"color"`
	Icon        string    `json:"icon" db:"icon"`
	Description string    `json:"description" db:"description"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// Tag представляет тег новости
type Tag struct {
	ID         int       `json:"id" db:"id"`
	Name       string    `json:"name" db:"name"`
	Slug       string    `json:"slug" db:"slug"`
	UsageCount int       `json:"usage_count" db:"usage_count"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// Country представляет страну
type Country struct {
	ID        int       `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Code      string    `json:"code" db:"code"`
	FlagEmoji string    `json:"flag_emoji" db:"flag_emoji"`
	IsActive  bool      `json:"is_active" db:"is_active"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// ParsingLog представляет лог парсинга RSS ленты
type ParsingLog struct {
	ID              int       `json:"id" db:"id"`
	SourceID        int       `json:"source_id" db:"source_id"`
	Status          string    `json:"status" db:"status"` // success, error, timeout
	NewsCount       int       `json:"news_count" db:"news_count"`
	ErrorMessage    string    `json:"error_message" db:"error_message"`
	ExecutionTimeMs int       `json:"execution_time_ms" db:"execution_time_ms"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`

	// Связанные данные
	Source *NewsSource `json:"source,omitempty" db:"-"`
}

// ParsedFeedItem представляет элемент, спарсенный из RSS ленты
type ParsedFeedItem struct {
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Content     string    `json:"content"`
	Link        string    `json:"link"`
	Author      string    `json:"author"`
	Published   time.Time `json:"published"`
	ImageURL    string    `json:"image_url"`
	GUID        string    `json:"guid"`
	Categories  []string  `json:"categories"`
}

// FeedParseResult представляет результат парсинга RSS ленты
type FeedParseResult struct {
	SourceID      int              `json:"source_id"`
	Items         []ParsedFeedItem `json:"items"`
	ParsedAt      time.Time        `json:"parsed_at"`
	Success       bool             `json:"success"`
	Error         string           `json:"error,omitempty"`
	ExecutionTime time.Duration    `json:"execution_time"`
}

// ParsingStats представляет статистику парсинга
type ParsingStats struct {
	TotalSources   int        `json:"total_sources"`
	ActiveSources  int        `json:"active_sources"`
	SuccessfulRuns int        `json:"successful_runs"`
	FailedRuns     int        `json:"failed_runs"`
	TotalNews      int        `json:"total_news"`
	NewsToday      int        `json:"news_today"`
	AvgParseTime   float64    `json:"avg_parse_time_ms"`
	LastParseTime  *time.Time `json:"last_parse_time"`
}

// StringArray представляет массив строк для работы с PostgreSQL
type StringArray []string

// Scan реализует интерфейс sql.Scanner для StringArray
func (a *StringArray) Scan(value interface{}) error {
	if value == nil {
		*a = nil
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, a)
	case string:
		return json.Unmarshal([]byte(v), a)
	default:
		return fmt.Errorf("cannot scan %T into StringArray", value)
	}
}

// Value реализует интерфейс driver.Valuer для StringArray
func (a StringArray) Value() (driver.Value, error) {
	if a == nil {
		return nil, nil
	}
	return json.Marshal(a)
}

// NewsFilter представляет фильтры для поиска новостей
type NewsFilter struct {
	SourceIDs   []int      `json:"source_ids,omitempty"`
	CategoryIDs []int      `json:"category_ids,omitempty"`
	CountryIDs  []int      `json:"country_ids,omitempty"`
	Keywords    string     `json:"keywords,omitempty"`
	DateFrom    *time.Time `json:"date_from,omitempty"`
	DateTo      *time.Time `json:"date_to,omitempty"`
	IsActive    *bool      `json:"is_active,omitempty"`
	Limit       int        `json:"limit,omitempty"`
	Offset      int        `json:"offset,omitempty"`
	SortBy      string     `json:"sort_by,omitempty"`    // published_at, relevance_score, created_at
	SortOrder   string     `json:"sort_order,omitempty"` // asc, desc
}

// Validation методы

// IsValid проверяет валидность новости
func (n *News) IsValid() bool {
	return len(n.Title) >= 10 && len(n.Title) <= 500 &&
		n.URL != "" &&
		n.SourceID > 0 &&
		!n.PublishedAt.IsZero()
}

// IsValid проверяет валидность источника новостей
func (ns *NewsSource) IsValid() bool {
	return ns.Name != "" &&
		ns.RSSURL != "" &&
		ns.CountryID > 0
}

// Методы для работы с временем

// IsRecent проверяет, является ли новость свежей (опубликована в последние 24 часа)
func (n *News) IsRecent() bool {
	return time.Since(n.PublishedAt) <= 24*time.Hour
}

// ShouldParse определяет, нужно ли парсить источник сейчас
func (ns *NewsSource) ShouldParse() bool {
	if !ns.IsActive {
		return false
	}

	if ns.LastParsedAt == nil {
		return true
	}

	interval := time.Duration(ns.ParseIntervalMinutes) * time.Minute
	return time.Since(*ns.LastParsedAt) >= interval
}

// GetParseInterval возвращает интервал парсинга для источника
func (ns *NewsSource) GetParseInterval() time.Duration {
	if ns.ParseIntervalMinutes <= 0 {
		return 10 * time.Minute // По умолчанию 10 минут
	}
	return time.Duration(ns.ParseIntervalMinutes) * time.Minute
}

// Константы для статусов парсинга
const (
	ParsingStatusSuccess = "success"
	ParsingStatusError   = "error"
	ParsingStatusTimeout = "timeout"
)

// Константы для сортировки
const (
	SortByPublishedAt    = "published_at"
	SortByRelevanceScore = "relevance_score"
	SortByCreatedAt      = "created_at"
	SortByViewCount      = "view_count"

	SortOrderAsc  = "asc"
	SortOrderDesc = "desc"
)

// Константы для категорий (соответствуют данным в БД)
const (
	CategorySports     = 1 // Спорт
	CategoryTechnology = 2 // Технологии
	CategoryPolitics   = 3 // Политика
	CategoryEconomics  = 4 // Экономика и финансы
	CategorySociety    = 5 // Общество
)

// GetCategoryName возвращает название категории по ID
func GetCategoryName(id int) string {
	categories := map[int]string{
		CategorySports:     "Спорт",
		CategoryTechnology: "Технологии",
		CategoryPolitics:   "Политика",
		CategoryEconomics:  "Экономика и финансы",
		CategorySociety:    "Общество",
	}

	if name, exists := categories[id]; exists {
		return name
	}
	return "Неизвестная категория"
}
