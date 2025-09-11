package services

import (
	"strings"

	"github.com/sirupsen/logrus"

	"news-parsing-service/internal/models"
)

// CategoryClassifier представляет классификатор категорий новостей
type CategoryClassifier struct {
	logger   *logrus.Logger
	keywords map[int][]string // категория -> ключевые слова
	// Многоязычные ключевые слова
	multilangKeywords map[int]map[string][]string // категория -> язык -> ключевые слова
}

// NewCategoryClassifier создает новый классификатор категорий
func NewCategoryClassifier(logger *logrus.Logger) *CategoryClassifier {
	// Определяем ключевые слова для каждой категории
	keywords := map[int][]string{
		models.CategoryRussia: {
			"политика", "правительство", "президент", "министр", "депутат", "парламент", "дума",
			"выборы", "партия", "оппозиция", "власть", "закон", "указ", "политический",
			"кремль", "администрация", "губернатор", "мэр", "чиновник", "бюджет", "налог",
			"санкции", "дипломатия", "посол", "внешняя политика", "международный", "геополитика",
			"путин", "зеленский", "байден", "макрон", "шольц", "нато", "ес", "оон", "оон",
			"государство", "государственный", "федеральный", "региональный", "муниципальный",
			"политик", "политическая", "политическое", "политические", "политиков",
			"голосование", "избиратель", "избирательный", "кандидат", "кампания",
			"коалиция", "фракция", "комитет", "комиссия", "служба", "агентство",
		},
		models.CategoryEconomy: {
			"экономика", "финансы", "рубль", "доллар", "евро", "валюта", "курс", "инфляция",
			"банк", "кредит", "инвестиции", "акции", "биржа", "нефть", "газ", "золото",
			"ВВП", "экспорт", "импорт", "торговля", "бизнес", "компания", "предприятие",
			"промышленность", "производство", "сельское хозяйство", "энергетика", "строительство",
		},
		// CategorySport удален, так как ID 4 используется для CategoryScienceTech
		models.CategoryScienceTech: {
			"технологии", "интернет", "компьютер", "смартфон", "телефон", "планшет", "ноутбук",
			"программное обеспечение", "приложение", "сайт", "социальные сети", "мессенджер",
			"искусственный интеллект", "машинное обучение", "блокчейн", "криптовалюта", "биткоин",
			"стартап", "IT", "цифровизация", "роботы", "автоматизация", "инновации", "разработка",
			"ИИ", "нейросеть", "чатбот", "чатгпт", "openai", "google", "apple", "microsoft",
			"android", "ios", "windows", "linux", "python", "javascript", "java", "c++",
			"веб", "мобильное приложение", "облачные технологии", "большие данные", "аналитика",
			"кибербезопасность", "хакер", "вирус", "антивирус", "защита данных", "приватность",
			"квантовый компьютер", "5g", "6g", "интернет вещей", "умный дом", "электромобиль",
			"тесла", "spacex", "метавселенная", "vr", "ar", "дополненная реальность",
			// Дополнительные ключевые слова для технологий
			"мобильный телефон", "гаджет", "устройство", "сервер", "база данных",
			"алгоритм", "нейросеть", "чат-бот", "цифровой", "онлайн", "офлайн",
			"веб-сайт", "интернет-магазин", "электронная коммерция", "эфириум", "NFT", "токен",
			"облако", "хостинг", "домен", "IP-адрес", "протокол", "API", "интерфейс",
			"операционная система", "браузер", "поисковик", "видеозвонок", "стриминг", "подкаст",
			"игровая консоль", "видеоигра", "киберспорт", "стример", "блогер",
			"электронная почта", "спам", "фишинг", "взлом", "уязвимость", "патч",
			"обновление", "версия", "релиз", "бета-тест", "альфа-тест", "дебаг",
			"код", "исходный код", "репозиторий", "git", "github", "gitlab",
			"фреймворк", "библиотека", "плагин", "расширение", "модуль", "компонент",
			"микросервис", "контейнер", "docker", "kubernetes", "devops", "ci/cd",
			"тестирование", "автотест", "юнит-тест", "интеграционный тест", "нагрузочное тестирование",
			"мониторинг", "логирование", "метрики", "аналитика", "дашборд", "отчет",
			"масштабирование", "производительность", "оптимизация", "кэширование", "CDN",
			"резервное копирование", "восстановление", "миграция", "деплой", "развертывание",
		},
		models.CategoryCulture: {
			"культура", "искусство", "театр", "кино", "фильм", "актер", "режиссер", "музыка",
			"концерт", "музыкант", "певец", "композитор", "выставка", "музей", "галерея",
			"картина", "художник", "скульптор", "литература", "книга", "писатель", "поэт",
			"фестиваль", "премия", "награда", "творчество", "культурный", "художественный",
		},
		models.CategorySociety: {
			"общество", "социальный", "население", "люди", "семья", "дети", "молодежь", "пенсионеры",
			"образование", "школа", "университет", "студент", "учитель", "профессор", "реформа",
			"социальная помощь", "пенсия", "пособие", "льготы", "жилье", "коммунальные услуги",
			"транспорт", "дороги", "инфраструктура", "благоустройство", "экология", "природа",
		},
		models.CategoryIncidents: {
			"происшествие", "авария", "катастрофа", "пожар", "взрыв", "наводнение", "землетрясение",
			"ураган", "смерч", "чрезвычайная ситуация", "спасательная операция", "эвакуация",
			"преступление", "кража", "грабеж", "убийство", "нападение", "арест", "полиция",
			"следствие", "суд", "приговор", "тюрьма", "ДТП", "столкновение", "авиакатастрофа",
		},
		models.CategoryHealth: {
			"здоровье", "медицина", "больница", "поликлиника", "врач", "доктор", "медсестра",
			"лечение", "операция", "диагноз", "болезнь", "заболевание", "эпидемия", "пандемия",
			"вирус", "инфекция", "вакцинация", "прививка", "лекарство", "препарат", "терапия",
			"реабилитация", "профилактика", "симптом", "здравоохранение", "медицинский",
		},
		models.CategoryEducation: {
			"образование", "школа", "университет", "институт", "колледж", "техникум", "детский сад",
			"учитель", "преподаватель", "профессор", "студент", "ученик", "учеба", "обучение",
			"экзамен", "тест", "оценка", "диплом", "аттестат", "стипендия", "учебник", "урок",
			"лекция", "семинар", "практика", "курсы", "образовательный", "педагогический",
		},
		models.CategoryWorld: {
			"международный", "зарубежный", "страна", "государство", "США", "Европа", "Китай",
			"Япония", "Индия", "Германия", "Франция", "Великобритания", "посольство", "консульство",
			"дипломат", "переговоры", "соглашение", "договор", "саммит", "визит", "встреча",
			"мировой", "глобальный", "иностранный", "зарубежье", "граница", "виза", "миграция",
		},
		models.CategoryBusiness: {
			"бизнес", "предпринимательство", "стартап", "компания", "корпорация", "фирма", "предприятие",
			"директор", "руководитель", "менеджер", "сотрудник", "работник", "зарплата", "доходы",
			"прибыль", "убытки", "продажи", "покупатель", "клиент", "услуга", "товар", "продукт",
			"рынок", "конкуренция", "монополия", "франшиза", "лицензия", "патент", "торговля",
		},
	}

	// Многоязычные ключевые слова
	multilangKeywords := map[int]map[string][]string{
		models.CategoryRussia: {
			"ru": {
				"политика", "правительство", "президент", "министр", "депутат", "парламент", "дума",
				"выборы", "партия", "оппозиция", "власть", "закон", "указ", "политический",
				"кремль", "администрация", "губернатор", "мэр", "чиновник", "бюджет", "налог",
				"санкции", "дипломатия", "посол", "внешняя политика", "международный", "геополитика",
				"путин", "зеленский", "байден", "макрон", "шольц", "нато", "ес", "оон",
				"государство", "государственный", "федеральный", "региональный", "муниципальный",
				"политик", "политическая", "политическое", "политические", "политиков",
				"голосование", "избиратель", "избирательный", "кандидат", "кампания",
				"коалиция", "фракция", "комитет", "комиссия", "служба", "агентство",
			},
			"en": {
				"politics", "government", "president", "minister", "parliament", "congress",
				"election", "party", "opposition", "power", "law", "decree", "political",
				"administration", "governor", "mayor", "official", "budget", "tax",
				"sanctions", "diplomacy", "ambassador", "foreign policy", "international", "geopolitics",
				"putin", "zelensky", "biden", "macron", "scholz", "nato", "eu", "un",
				"state", "federal", "regional", "municipal", "politician", "political",
				"voting", "voter", "candidate", "campaign", "coalition", "faction",
				"committee", "commission", "service", "agency", "democracy", "republic",
			},
		},
		models.CategoryScienceTech: {
			"ru": {
				"технологии", "интернет", "компьютер", "смартфон", "телефон", "планшет", "ноутбук",
				"программное обеспечение", "приложение", "сайт", "социальные сети", "мессенджер",
				"искусственный интеллект", "машинное обучение", "блокчейн", "криптовалюта", "биткоин",
				"стартап", "IT", "цифровизация", "роботы", "автоматизация", "инновации", "разработка",
				"ИИ", "нейросеть", "чатбот", "чатгпт", "openai", "google", "apple", "microsoft",
				"android", "ios", "windows", "linux", "python", "javascript", "java", "c++",
				"веб", "мобильное приложение", "облачные технологии", "большие данные", "аналитика",
				"кибербезопасность", "хакер", "вирус", "антивирус", "защита данных", "приватность",
				"квантовый компьютер", "5g", "6g", "интернет вещей", "умный дом", "электромобиль",
				"тесла", "spacex", "метавселенная", "vr", "ar", "дополненная реальность",
			},
			"en": {
				"technology", "internet", "computer", "smartphone", "phone", "tablet", "laptop",
				"software", "application", "app", "website", "social media", "messenger",
				"artificial intelligence", "machine learning", "blockchain", "cryptocurrency", "bitcoin",
				"startup", "IT", "digitalization", "robots", "automation", "innovation", "development",
				"AI", "neural network", "chatbot", "chatgpt", "openai", "google", "apple", "microsoft",
				"android", "ios", "windows", "linux", "python", "javascript", "java", "c++",
				"web", "mobile app", "cloud technology", "big data", "analytics",
				"cybersecurity", "hacker", "virus", "antivirus", "data protection", "privacy",
				"quantum computer", "5g", "6g", "internet of things", "smart home", "electric car",
				"tesla", "spacex", "metaverse", "vr", "ar", "augmented reality",
			},
		},
		models.CategoryEconomy: {
			"ru": {
				"экономика", "финансы", "рубль", "доллар", "евро", "валюта", "курс", "инфляция",
				"банк", "кредит", "инвестиции", "акции", "биржа", "нефть", "газ", "золото",
				"ВВП", "экспорт", "импорт", "торговля", "бизнес", "компания", "предприятие",
				"промышленность", "производство", "сельское хозяйство", "энергетика", "строительство",
			},
			"en": {
				"economy", "finance", "ruble", "dollar", "euro", "currency", "rate", "inflation",
				"bank", "credit", "investment", "stocks", "exchange", "oil", "gas", "gold",
				"GDP", "export", "import", "trade", "business", "company", "enterprise",
				"industry", "production", "agriculture", "energy", "construction",
			},
		},
		// CategorySport удален из multilangKeywords
	}

	return &CategoryClassifier{
		logger:            logger,
		keywords:          keywords,
		multilangKeywords: multilangKeywords,
	}
}

// ClassifyNews классифицирует новость по категории
func (c *CategoryClassifier) ClassifyNews(title, description string, categories []string) *int {
	// Сначала пробуем определить категорию из RSS категорий
	if rssCategoryID := c.mapRSSCategoryToInternal(categories); rssCategoryID != nil {
		c.logger.WithFields(logrus.Fields{
			"title":              truncateForLog(title, 50),
			"rss_categories":     categories,
			"mapped_category_id": *rssCategoryID,
		}).Info("Category determined from RSS categories")
		return rssCategoryID
	}

	// Если RSS категории не помогли, используем классификатор по ключевым словам
	// Объединяем весь текст для анализа
	text := strings.ToLower(title + " " + description + " " + strings.Join(categories, " "))

	// Определяем язык текста
	detectedLang := c.detectLanguage(text)

	// Подсчитываем совпадения ключевых слов для каждой категории
	categoryScores := make(map[int]float64)

	// Используем многоязычные ключевые слова если доступны
	if _, exists := c.multilangKeywords[models.CategoryRussia]; exists {
		// Используем многоязычную классификацию
		for categoryID, langKeywords := range c.multilangKeywords {
			score := 0.0

			// Проверяем ключевые слова для определенного языка
			if keywords, langExists := langKeywords[detectedLang]; langExists {
				for _, keyword := range keywords {
					keywordLower := strings.ToLower(keyword)
					if strings.Contains(text, keywordLower) {
						weight := c.calculateKeywordWeight(keyword, text, title, description)
						score += weight
					}
				}
			}

			// Также проверяем русские ключевые слова как fallback
			if detectedLang != "ru" {
				if keywords, exists := c.keywords[categoryID]; exists {
					for _, keyword := range keywords {
						keywordLower := strings.ToLower(keyword)
						if strings.Contains(text, keywordLower) {
							weight := c.calculateKeywordWeight(keyword, text, title, description) * 0.5 // Меньший вес для fallback
							score += weight
						}
					}
				}
			}

			if score > 0 {
				categoryScores[categoryID] = score
			}
		}
	} else {
		// Fallback к старому алгоритму
		for categoryID, keywords := range c.keywords {
			score := 0.0
			for _, keyword := range keywords {
				keywordLower := strings.ToLower(keyword)
				if strings.Contains(text, keywordLower) {
					weight := c.calculateKeywordWeight(keyword, text, title, description)
					score += weight
				}
			}
			if score > 0 {
				categoryScores[categoryID] = score
			}
		}
	}

	// Если нет совпадений, возвращаем nil (категория не определена)
	if len(categoryScores) == 0 {
		return nil
	}

	// Находим категорию с максимальным количеством совпадений
	var bestCategory int
	var maxScore float64

	for categoryID, score := range categoryScores {
		if score > maxScore {
			maxScore = score
			bestCategory = categoryID
		}
	}

	// Минимальный порог для уверенности в классификации
	minThreshold := 1.0
	if maxScore < minThreshold {
		c.logger.WithFields(logrus.Fields{
			"title":         truncateForLog(title, 50),
			"detected_lang": detectedLang,
			"max_score":     maxScore,
			"threshold":     minThreshold,
		}).Debug("Category score below threshold, returning nil")
		return nil
	}

	// Логируем результат классификации для отладки
	c.logger.WithFields(logrus.Fields{
		"title":         truncateForLog(title, 50),
		"description":   truncateForLog(description, 100),
		"detected_lang": detectedLang,
		"category_id":   bestCategory,
		"score":         maxScore,
		"all_scores":    categoryScores,
		"threshold":     minThreshold,
	}).Info("Classified news category")

	return &bestCategory
}

// detectLanguage определяет язык текста по простым эвристикам
func (c *CategoryClassifier) detectLanguage(text string) string {
	// Простая эвристика для определения языка
	// Подсчитываем количество кириллических и латинских символов

	cyrillicCount := 0
	latinCount := 0

	for _, char := range text {
		if char >= 'а' && char <= 'я' || char >= 'А' && char <= 'Я' {
			cyrillicCount++
		} else if char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' {
			latinCount++
		}
	}

	// Если больше кириллических символов - русский
	if cyrillicCount > latinCount {
		return "ru"
	}

	// Иначе английский (можно расширить для других языков)
	return "en"
}

// calculateKeywordWeight вычисляет вес ключевого слова
func (c *CategoryClassifier) calculateKeywordWeight(keyword, fullText, title, description string) float64 {
	keywordLower := strings.ToLower(keyword)
	weight := 1.0

	// Более длинные ключевые слова имеют больший вес
	if len(keyword) > 10 {
		weight *= 1.5
	} else if len(keyword) > 5 {
		weight *= 1.2
	}

	// Ключевые слова в заголовке имеют больший вес (самое важное)
	if strings.Contains(strings.ToLower(title), keywordLower) {
		weight *= 3.0 // Увеличиваем с 2.0 до 3.0
	}

	// Ключевые слова в описании имеют средний вес
	if strings.Contains(strings.ToLower(description), keywordLower) {
		weight *= 1.5 // Увеличиваем с 1.3 до 1.5
	}

	// Подсчитываем количество вхождений
	occurrences := strings.Count(fullText, keywordLower)
	if occurrences > 1 {
		weight *= float64(occurrences) * 0.3 // Уменьшаем влияние повторений
	}

	// Дополнительный вес для специфичных технологических терминов
	techTerms := []string{"ИИ", "искусственный интеллект", "нейросеть", "чатгпт", "openai",
		"машинное обучение", "блокчейн", "криптовалюта", "биткоин", "NFT", "метавселенная",
		"кибербезопасность", "хакер", "вирус", "антивирус", "квантовый компьютер", "5g", "6g",
		"интернет вещей", "умный дом", "электромобиль", "тесла", "spacex", "vr", "ar"}

	for _, techTerm := range techTerms {
		if strings.Contains(keywordLower, strings.ToLower(techTerm)) {
			weight *= 1.5
			break
		}
	}

	return weight
}

// GetCategoryKeywords возвращает ключевые слова для категории
func (c *CategoryClassifier) GetCategoryKeywords(categoryID int) []string {
	if keywords, exists := c.keywords[categoryID]; exists {
		return keywords
	}
	return nil
}

// AddKeywords добавляет ключевые слова для категории
func (c *CategoryClassifier) AddKeywords(categoryID int, keywords []string) {
	if existingKeywords, exists := c.keywords[categoryID]; exists {
		c.keywords[categoryID] = append(existingKeywords, keywords...)
	} else {
		c.keywords[categoryID] = keywords
	}

	c.logger.WithFields(logrus.Fields{
		"category_id":    categoryID,
		"added_keywords": keywords,
		"total_keywords": len(c.keywords[categoryID]),
	}).Debug("Added keywords to category")
}

// RemoveKeywords удаляет ключевые слова из категории
func (c *CategoryClassifier) RemoveKeywords(categoryID int, keywordsToRemove []string) {
	if keywords, exists := c.keywords[categoryID]; exists {
		var filteredKeywords []string
		for _, keyword := range keywords {
			shouldRemove := false
			for _, removeKeyword := range keywordsToRemove {
				if strings.EqualFold(keyword, removeKeyword) {
					shouldRemove = true
					break
				}
			}
			if !shouldRemove {
				filteredKeywords = append(filteredKeywords, keyword)
			}
		}
		c.keywords[categoryID] = filteredKeywords

		c.logger.WithFields(logrus.Fields{
			"category_id":        categoryID,
			"removed_keywords":   keywordsToRemove,
			"remaining_keywords": len(filteredKeywords),
		}).Debug("Removed keywords from category")
	}
}

// GetAllCategories возвращает все доступные категории с их ключевыми словами
func (c *CategoryClassifier) GetAllCategories() map[int][]string {
	result := make(map[int][]string)
	for categoryID, keywords := range c.keywords {
		// Создаем копию слайса, чтобы избежать изменений извне
		result[categoryID] = make([]string, len(keywords))
		copy(result[categoryID], keywords)
	}
	return result
}

// UpdateKeywords полностью заменяет ключевые слова для категории
func (c *CategoryClassifier) UpdateKeywords(categoryID int, keywords []string) {
	c.keywords[categoryID] = make([]string, len(keywords))
	copy(c.keywords[categoryID], keywords)

	c.logger.WithFields(logrus.Fields{
		"category_id":    categoryID,
		"keywords_count": len(keywords),
	}).Info("Updated keywords for category")
}

// ClassifyWithConfidence классифицирует новость с указанием уверенности
func (c *CategoryClassifier) ClassifyWithConfidence(title, description string, categories []string) (categoryID *int, confidence float64) {
	text := strings.ToLower(title + " " + description + " " + strings.Join(categories, " "))

	categoryScores := make(map[int]int)
	totalWords := len(strings.Fields(text))

	for catID, keywords := range c.keywords {
		score := 0
		for _, keyword := range keywords {
			if strings.Contains(text, strings.ToLower(keyword)) {
				score++
			}
		}
		if score > 0 {
			categoryScores[catID] = score
		}
	}

	if len(categoryScores) == 0 {
		return nil, 0.0
	}

	// Находим лучшую категорию
	var bestCategory int
	var maxScore int

	for catID, score := range categoryScores {
		if score > maxScore {
			maxScore = score
			bestCategory = catID
		}
	}

	// Вычисляем уверенность как отношение совпадений к общему количеству слов
	confidence = float64(maxScore) / float64(totalWords)
	if confidence > 1.0 {
		confidence = 1.0
	}

	// Минимальный порог уверенности
	if confidence < 0.1 {
		return nil, confidence
	}

	return &bestCategory, confidence
}

// mapRSSCategoryToInternal маппит RSS категории на внутренние категории
func (c *CategoryClassifier) mapRSSCategoryToInternal(rssCategories []string) *int {
	if len(rssCategories) == 0 {
		return nil
	}

	// Маппинг RSS категорий на внутренние ID категорий
	categoryMapping := map[string]int{
		// Политика (перенаправляем на Россию)
		"политика":      models.CategoryRussia,
		"politics":      models.CategoryRussia,
		"россия":        models.CategoryRussia,
		"russia":        models.CategoryRussia,
		"украина":       models.CategoryExUSSR,
		"ukraine":       models.CategoryExUSSR,
		"сша":           models.CategoryWorld,
		"usa":           models.CategoryWorld,
		"европа":        models.CategoryWorld,
		"europe":        models.CategoryWorld,
		"кремль":        models.CategoryRussia,
		"президент":     models.CategoryRussia,
		"правительство": models.CategoryRussia,
		"парламент":     models.CategoryRussia,
		"выборы":        models.CategoryRussia,
		"выборы 2025":   models.CategoryRussia,

		// Экономика
		"экономика":  models.CategoryEconomy,
		"economy":    models.CategoryEconomy,
		"бизнес":     models.CategoryEconomy,
		"business":   models.CategoryEconomy,
		"финансы":    models.CategoryEconomy,
		"finance":    models.CategoryEconomy,
		"рынок":      models.CategoryEconomy,
		"market":     models.CategoryEconomy,
		"валюты":     models.CategoryEconomy,
		"currency":   models.CategoryEconomy,
		"нефть":      models.CategoryEconomy,
		"oil":        models.CategoryEconomy,
		"газ":        models.CategoryEconomy,
		"gas":        models.CategoryEconomy,
		"банк":       models.CategoryEconomy,
		"bank":       models.CategoryEconomy,
		"инвестиции": models.CategoryEconomy,
		"investment": models.CategoryEconomy,

		// Спорт
		"спорт":        models.CategorySportNew,
		"sport":        models.CategorySportNew,
		"футбол":       models.CategorySportNew,
		"football":     models.CategorySportNew,
		"хоккей":       models.CategorySportNew,
		"hockey":       models.CategorySportNew,
		"нхл":          models.CategorySportNew,
		"nhl":          models.CategorySportNew,
		"баскетбол":    models.CategorySportNew,
		"basketball":   models.CategorySportNew,
		"теннис":       models.CategorySportNew,
		"tennis":       models.CategorySportNew,
		"бокс":         models.CategorySportNew,
		"boxing":       models.CategorySportNew,
		"олимпиада":    models.CategorySportNew,
		"olympics":     models.CategorySportNew,
		"чемпионат":    models.CategorySportNew,
		"championship": models.CategorySportNew,

		// Технологии
		"технологии": models.CategoryScienceTech,
		"technology": models.CategoryScienceTech,
		"техника":    models.CategoryScienceTech,
		"tech":       models.CategoryScienceTech,
		"it":         models.CategoryScienceTech,
		"интернет":   models.CategoryScienceTech,
		"internet":   models.CategoryScienceTech,
		"компьютер":  models.CategoryScienceTech,
		"computer":   models.CategoryScienceTech,
		"смартфон":   models.CategoryScienceTech,
		"smartphone": models.CategoryScienceTech,
		"ии":         models.CategoryScienceTech,
		"ai":         models.CategoryScienceTech,
		"искусственный интеллект": models.CategoryScienceTech,
		"artificial intelligence": models.CategoryScienceTech,
		"робот":                   models.CategoryScienceTech,
		"robot":                   models.CategoryScienceTech,
		"автомобиль":              models.CategoryScienceTech,
		"car":                     models.CategoryScienceTech,
		"электромобиль":           models.CategoryScienceTech,
		"electric car":            models.CategoryScienceTech,

		// Общество
		"общество":    models.CategorySociety,
		"society":     models.CategorySociety,
		"социальный":  models.CategorySociety,
		"social":      models.CategorySociety,
		"образование": models.CategorySociety,
		"education":   models.CategorySociety,
		"школа":       models.CategorySociety,
		"school":      models.CategorySociety,
		"университет": models.CategorySociety,
		"university":  models.CategorySociety,
		"студент":     models.CategorySociety,
		"student":     models.CategorySociety,
		"стипендия":   models.CategorySociety,
		"scholarship": models.CategorySociety,
		"пенсия":      models.CategorySociety,
		"pension":     models.CategorySociety,
		"семья":       models.CategorySociety,
		"family":      models.CategorySociety,
		"дети":        models.CategorySociety,
		"children":    models.CategorySociety,

		// Культура
		"культура":   models.CategoryCulture,
		"culture":    models.CategoryCulture,
		"искусство":  models.CategoryCulture,
		"art":        models.CategoryCulture,
		"кино":       models.CategoryCulture,
		"cinema":     models.CategoryCulture,
		"фильм":      models.CategoryCulture,
		"movie":      models.CategoryCulture,
		"музыка":     models.CategoryCulture,
		"music":      models.CategoryCulture,
		"театр":      models.CategoryCulture,
		"theater":    models.CategoryCulture,
		"литература": models.CategoryCulture,
		"literature": models.CategoryCulture,
		"книга":      models.CategoryCulture,
		"book":       models.CategoryCulture,

		// Здоровье
		"здоровье":  models.CategoryHealth,
		"health":    models.CategoryHealth,
		"медицина":  models.CategoryHealth,
		"medicine":  models.CategoryHealth,
		"больница":  models.CategoryHealth,
		"hospital":  models.CategoryHealth,
		"врач":      models.CategoryHealth,
		"doctor":    models.CategoryHealth,
		"лечение":   models.CategoryHealth,
		"treatment": models.CategoryHealth,
		"болезнь":   models.CategoryHealth,
		"disease":   models.CategoryHealth,
		"вирус":     models.CategoryHealth,
		"virus":     models.CategoryHealth,
		"вакцина":   models.CategoryHealth,
		"vaccine":   models.CategoryHealth,

		// Происшествия
		"происшествие":     models.CategoryIncidents,
		"incident":         models.CategoryIncidents,
		"авария":           models.CategoryIncidents,
		"accident":         models.CategoryIncidents,
		"катастрофа":       models.CategoryIncidents,
		"disaster":         models.CategoryIncidents,
		"пожар":            models.CategoryIncidents,
		"fire":             models.CategoryIncidents,
		"взрыв":            models.CategoryIncidents,
		"explosion":        models.CategoryIncidents,
		"преступление":     models.CategoryIncidents,
		"crime":            models.CategoryIncidents,
		"дтп":              models.CategoryIncidents,
		"traffic accident": models.CategoryIncidents,
	}

	// Проверяем каждую RSS категорию
	for _, rssCategory := range rssCategories {
		rssCategoryLower := strings.ToLower(strings.TrimSpace(rssCategory))

		// Прямое совпадение
		if categoryID, exists := categoryMapping[rssCategoryLower]; exists {
			return &categoryID
		}

		// Проверяем частичные совпадения
		for mappingKey, categoryID := range categoryMapping {
			if strings.Contains(rssCategoryLower, mappingKey) || strings.Contains(mappingKey, rssCategoryLower) {
				return &categoryID
			}
		}
	}

	return nil
}
