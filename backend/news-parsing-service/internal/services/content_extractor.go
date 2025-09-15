package services

import (
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/go-shiori/go-readability"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/html/charset"
)

// ContentExtractor извлекает основной текст статей с веб-страниц.
type ContentExtractor struct {
	logger *logrus.Logger
	config *ContentExtractorConfig
	client *http.Client
}

// ContentExtractorConfig конфигурация извлекателя контента.
type ContentExtractorConfig struct {
	RequestTimeout time.Duration
	UserAgent      string
	EnableFullText bool
}

// NewContentExtractor создает новый извлекатель контента.
// Если config == nil, применяются значения по умолчанию.
func NewContentExtractor(logger *logrus.Logger, config *ContentExtractorConfig) (*ContentExtractor, error) {
	if config == nil {
		config = &ContentExtractorConfig{
			RequestTimeout: 30 * time.Second,
			UserAgent:      "Mozilla/5.0 (compatible; GoReadabilityBot/1.0)",
			EnableFullText: true,
		}
	}
	if logger == nil {
		logger = logrus.New()
	}

	// Транспорт с дополнительными таймаутами.
	tr := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		IdleConnTimeout:       30 * time.Second,
	}

	client := &http.Client{
		Timeout:   config.RequestTimeout, // общий дедлайн на запрос
		Transport: tr,
		// По умолчанию редиректы разрешены; можно ограничить при желании:
		// CheckRedirect: func(req *http.Request, via []*http.Request) error {
		// 	if len(via) >= 5 { return fmt.Errorf("stopped after 5 redirects") }
		// 	return nil
		// },
	}

	logger.Info("Creating content extractor with go-readability")

	return &ContentExtractor{
		logger: logger,
		config: config,
		client: client,
	}, nil
}

// ExtractFullContent извлекает основной текст статьи с помощью go-readability.
// Возвращает очищенный текст или ошибку.
func (e *ContentExtractor) ExtractFullContent(ctx context.Context, pageURL string) (string, error) {
	if !e.config.EnableFullText {
		return "", nil
	}
	if !e.IsValidURL(pageURL) {
		return "", fmt.Errorf("invalid URL: %s", pageURL)
	}

	e.logger.WithField("url", pageURL).Debug("Extracting full content from URL with go-readability")

	// Создаем запрос
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	// ВАЖНО: не ставим Accept-Encoding вручную — stdlib сама разожмет gzip.
	req.Header.Set("User-Agent", e.config.UserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "ru-RU,ru;q=0.9,en;q=0.8")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Cache-Control", "no-cache")

	// Выполняем запрос
	resp, err := e.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	// Проверяем статус
	if resp.StatusCode == http.StatusNotFound {
		e.logger.WithField("url", pageURL).Debug("Article not found (404)")
		return "", fmt.Errorf("article not found: HTTP 404")
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	// Проверяем тип содержимого (ожидаем HTML)
	if ct := resp.Header.Get("Content-Type"); ct != "" && !strings.Contains(strings.ToLower(ct), "text/html") {
		return "", fmt.Errorf("unsupported content type: %s", ct)
	}

	// 1) Декодируем тело по Content-Encoding (gzip/br/deflate/identity)
	decodedBody, err := decodeBody(resp)
	if err != nil {
		return "", fmt.Errorf("failed to decode body: %w", err)
	}
	// Закрывать будем decodedBody, если это не исходный resp.Body:
	defer func() {
		if c, ok := decodedBody.(io.Closer); ok && decodedBody != resp.Body {
			_ = c.Close()
		}
	}()

	// 2) Перекодируем в UTF-8 с учетом заголовков и <meta charset=...>
	utf8Body, err := toUTF8Reader(decodedBody, resp)
	if err != nil {
		return "", fmt.Errorf("failed to convert to UTF-8: %w", err)
	}

	// 3) (опционально) ограничим размер входного потока
	// utf8Body = io.LimitReader(utf8Body, 5<<20) // 5MB

	// 4) Читаем через go-readability
	parsedURL, err := url.Parse(pageURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse URL: %w", err)
	}

	e.logger.WithField("url", pageURL).Debug("Calling go-readability.FromReader")
	article, err := readability.FromReader(utf8Body, parsedURL)
	if err != nil {
		e.logger.WithError(err).Error("go-readability failed to extract content")
		return "", fmt.Errorf("failed to extract content with readability: %w", err)
	}

	// Логируем
	e.logger.WithFields(logrus.Fields{
		"url":         pageURL,
		"title":       article.Title,
		"text_length": len([]rune(article.TextContent)),
		"excerpt":     article.Excerpt,
	}).Debug("go-readability successfully extracted content")

	// Проверяем длину (порог можно ослабить/изменить)
	if len([]rune(article.TextContent)) < 80 {
		return "", fmt.Errorf("insufficient content extracted: %d characters", len([]rune(article.TextContent)))
	}

	// Чистим и ограничиваем безопасно по рунам
	content := e.cleanText(article.TextContent)

	// Дополнительная очистка для go-readability
	content = e.cleanReadabilityContent(content, pageURL, article.Title)

	e.logger.WithFields(logrus.Fields{
		"url":            pageURL,
		"title":          article.Title,
		"content_length": len([]rune(content)),
		"excerpt":        article.Excerpt,
	}).Debug("Successfully extracted and cleaned content")

	return content, nil
}

// decodeBody распаковывает тело ответа согласно Content-Encoding.
func decodeBody(resp *http.Response) (io.ReadCloser, error) {
	body := resp.Body
	ce := strings.ToLower(strings.TrimSpace(resp.Header.Get("Content-Encoding")))

	switch ce {
	case "":
		// Пусто — stdlib могла уже разжать gzip автоматически (Transparent Decompression).
		return body, nil
	case "gzip":
		gr, err := gzip.NewReader(body)
		if err != nil {
			return nil, err
		}
		return gr, nil
	case "br":
		// Brotli через внешний пакет.
		return io.NopCloser(brotli.NewReader(body)), nil
	case "deflate":
		// Сначала пробуем zlib, если не вышло — сырой deflate.
		zr, err := zlib.NewReader(body)
		if err == nil {
			return zr, nil
		}
		fr := flate.NewReader(body)
		return fr, nil
	default:
		// Неожиданный кодек — отдаем как есть.
		return body, nil
	}
}

// toUTF8Reader возвращает Reader в UTF-8, учитывая Content-Type/charset и meta-charset.
func toUTF8Reader(r io.Reader, resp *http.Response) (io.Reader, error) {
	ctHeader := resp.Header.Get("Content-Type")
	_, params, _ := mime.ParseMediaType(ctHeader)
	if cs, ok := params["charset"]; ok && cs != "" {
		utf8r, err := charset.NewReaderLabel(strings.ToLower(cs), r)
		if err == nil {
			return utf8r, nil
		}
	}
	// Автоопределение по байтам/мета-тегам
	utf8r, err := charset.NewReader(r, ctHeader)
	if err == nil {
		return utf8r, nil
	}
	// Если уже UTF-8 или определить не удалось — возвращаем как есть.
	return r, nil
}

// cleanReadabilityContent дополнительно очищает контент, извлеченный go-readability
func (e *ContentExtractor) cleanReadabilityContent(content, pageURL, title string) string {
	// Удаляем URL страницы из начала контента
	content = strings.TrimPrefix(content, pageURL)

	// Удаляем повторяющиеся заголовки (более точное регулярное выражение)
	titlePattern := regexp.MustCompile(`(` + regexp.QuoteMeta(title) + `)+`)
	content = titlePattern.ReplaceAllString(content, title)

	// Удаляем метаданные и теги в конце
	content = regexp.MustCompile(`\s*РИА Новости, \d{2}\.\d{2}\.\d{4}.*$`).ReplaceAllString(content, "")
	content = regexp.MustCompile(`\s*-\s*РИА Новости.*$`).ReplaceAllString(content, "")

	// Удаляем повторяющиеся даты и временные метки
	content = regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\+\d{2}:\d{2}`).ReplaceAllString(content, "")
	content = regexp.MustCompile(`\d{2}\.\d{2}\.\d{4}\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\+\d{2}:\d{2}`).ReplaceAllString(content, "")

	// Удаляем теги и метаданные в конце
	content = regexp.MustCompile(`\s*[а-яё\s]+(россия|москва|костромская область|татьяна москалькова|федеральная служба исполнения наказаний|фсин россии|единый день голосования|2025).*$`).ReplaceAllString(content, "")

	// Удаляем URL изображений
	content = regexp.MustCompile(`https://[^\s]+\.(jpg|jpeg|png|gif|webp)[^\s]*`).ReplaceAllString(content, "")

	// Удаляем "MOCKBА" и подобные артефакты
	content = regexp.MustCompile(`MOCKB[А-Яа-я]+`).ReplaceAllString(content, "")

	// Удаляем дублирующиеся заголовки в начале
	content = regexp.MustCompile(`^(`+regexp.QuoteMeta(title)+`\s*)+`).ReplaceAllString(content, title+" ")

	// НОВОЕ: Удаляем связанные новости и рекламные блоки
	// Удаляем блоки, которые начинаются с "Читать далее" или "Читать полностью"
	content = regexp.MustCompile(`\s*Читать далее.*$`).ReplaceAllString(content, "")
	content = regexp.MustCompile(`\s*Читать полностью.*$`).ReplaceAllString(content, "")

	// Удаляем блоки, которые выглядят как отдельные новости (содержат имена людей и "вспомнил", "заявил" и т.д.)
	content = regexp.MustCompile(`\s*[А-Яа-яё]+ [А-Яа-яё]+ [А-Яа-яё]+ (вспомнил|заявил|отметил|сообщил|рассказал|подчеркнул).*$`).ReplaceAllString(content, "")

	// Удаляем блоки с футбольными командами и игроками (часто реклама)
	content = regexp.MustCompile(`\s*[А-Яа-яё]+ [А-Яа-яё]+ «[А-Яа-яё]+» [А-Яа-яё]+.*$`).ReplaceAllString(content, "")

	// Удаляем блоки, которые содержат "сборной России" (часто реклама)
	content = regexp.MustCompile(`\s*[А-Яа-яё\s]+сборной России[А-Яа-яё\s]*.*$`).ReplaceAllString(content, "")

	// Удаляем блоки с клубами и командами
	content = regexp.MustCompile(`\s*[А-Яа-яё\s]+«[А-Яа-яё]+»[А-Яа-яё\s]*.*$`).ReplaceAllString(content, "")

	// Удаляем блоки, которые содержат "интересуются" (часто реклама)
	content = regexp.MustCompile(`\s*[А-Яа-яё\s]+интересуются[А-Яа-яё\s]*.*$`).ReplaceAllString(content, "")

	// Удаляем блоки, которые содержат "клубе" (часто реклама)
	content = regexp.MustCompile(`\s*[А-Яа-яё\s]+клубе[А-Яа-яё\s]*.*$`).ReplaceAllString(content, "")

	// Очищаем от лишних пробелов
	content = strings.TrimSpace(content)

	return content
}

// cleanText приводит текст к аккуратному виду и безопасно обрезает по рунам.
func (e *ContentExtractor) cleanText(text string) string {
	// Приводим переводы строк к \n и обрезаем края
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.TrimSpace(text)

	// Схлопываем длинные последовательности пробелов/табов в один пробел,
	// но не трогаем переводы строк
	reSpaces := regexp.MustCompile(`[ \t]+`)
	text = reSpaces.ReplaceAllString(text, " ")

	// (Опционально) Схлопнуть 3+ пустых строк до одной пустой строки
	reGaps := regexp.MustCompile(`\n{3,}`)
	text = reGaps.ReplaceAllString(text, "\n\n")

	// Ограничиваем длину контента по рунам (без порчи UTF-8)
	const maxRunes = 10000 // ~10K символов
	runes := []rune(text)
	if len(runes) > maxRunes {
		text = string(runes[:maxRunes]) + "..."
	}
	return text
}

// Close закрывает ресурсы (сейчас нечего закрывать, оставлено для совместимости).
func (e *ContentExtractor) Close() error {
	return nil
}

// IsValidURL выполняет базовую проверку URL для извлечения контента.
func (e *ContentExtractor) IsValidURL(pageURL string) bool {
	// Простая проверка схемы
	if !strings.HasPrefix(pageURL, "http://") && !strings.HasPrefix(pageURL, "https://") {
		return false
	}

	// Исключаем некоторые типы URL
	excludePatterns := []string{
		".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx",
		".jpg", ".jpeg", ".png", ".gif", ".svg", ".webp",
		".mp3", ".mp4", ".avi", ".mov", ".wmv",
		".zip", ".rar", ".7z", ".tar", ".gz",
	}
	urlLower := strings.ToLower(pageURL)
	for _, pattern := range excludePatterns {
		if strings.Contains(urlLower, pattern) {
			return false
		}
	}
	return true
}
