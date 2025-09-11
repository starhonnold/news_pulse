package services

import (
	"strings"

	"github.com/sirupsen/logrus"
)

// CountryDetector представляет детектор стран в новостях
type CountryDetector struct {
	logger   *logrus.Logger
	keywords map[string][]string // код страны -> ключевые слова
}

// NewCountryDetector создает новый детектор стран
func NewCountryDetector(logger *logrus.Logger) *CountryDetector {
	// Определяем ключевые слова для каждой страны
	keywords := map[string][]string{
		"ru": {
			"россия", "российский", "российская", "российское", "российские",
			"москва", "московский", "московская", "московское", "московские",
			"санкт-петербург", "петербург", "ленинград", "питер",
			"путин", "медведев", "мишустин", "лавров", "шойгу",
			"кремль", "дума", "совет федерации", "правительство рф",
			"рубль", "российский рубль", "цб рф", "центральный банк россии",
			"газпром", "роснефть", "лукойл", "сбербанк", "втб",
			"российская армия", "вс рф", "вооруженные силы россии",
			"российская федерация", "рф", "россия",
		},
		"us": {
			"сша", "америка", "американский", "американская", "американское", "американские",
			"вашингтон", "нью-йорк", "лос-анджелес", "чикаго", "хьюстон",
			"байден", "трамп", "обма", "буш", "клинтон",
			"белый дом", "конгресс", "сенат", "палата представителей",
			"доллар", "американский доллар", "фрс", "федеральная резервная система",
			"apple", "microsoft", "google", "amazon", "tesla", "meta",
			"американская армия", "пентагон", "цру", "фбр", "нса",
			"соединенные штаты", "соединенные штаты америки", "usa", "us",
		},
		"ua": {
			"украина", "украинский", "украинская", "украинское", "украинские",
			"киев", "киевский", "киевская", "киевское", "киевские",
			"харьков", "одесса", "днепр", "донецк", "львов",
			"зеленский", "порошенко", "яценюк", "кулеба", "резунков",
			"верховная рада", "рада", "правительство украины",
			"гривна", "украинская гривна", "нбу", "национальный банк украины",
			"всу", "вооруженные силы украины", "украинская армия",
			"украинская республика", "уа", "ukraine", "ua",
		},
		"de": {
			"германия", "немецкий", "немецкая", "немецкое", "немецкие",
			"берлин", "мюнхен", "гамбург", "франкфурт", "кёльн",
			"шольц", "меркель", "шредер", "кол", "байербок",
			"бундестаг", "бундесрат", "правительство германии",
			"евро", "немецкая марка", "бундесбанк",
			"bmw", "mercedes", "volkswagen", "audi", "siemens",
			"бундесвер", "немецкая армия", "фрг", "germany", "de",
		},
		"fr": {
			"франция", "французский", "французская", "французское", "французские",
			"париж", "лион", "марсель", "тулуза", "ницца",
			"макрон", "олланд", "саркози", "ширак", "миттеран",
			"национальное собрание", "сенат", "правительство франции",
			"евро", "французский франк", "банк франции",
			"renault", "peugeot", "citroen", "airbus", "total",
			"французская армия", "иностранный легион", "фр", "france", "fr",
		},
		"gb": {
			"великобритания", "британский", "британская", "британское", "британские",
			"лондон", "манчестер", "ливерпуль", "бirmingham", "лидс",
			"сунак", "джонсон", "мэй", "кэмерон", "блер",
			"палата общин", "палата лордов", "правительство великобритании",
			"фунт", "британский фунт", "банк англии",
			"bp", "shell", "astrazeneca", "glaxosmithkline", "unilever",
			"британская армия", "королевские военно-воздушные силы", "снг", "uk", "gb",
		},
		"cn": {
			"китай", "китайский", "китайская", "китайское", "китайские",
			"пекин", "шанхай", "гуанчжоу", "шэньчжэнь", "чунцин",
			"си цзиньпин", "ху цзиньтао", "цзян цзэминь", "ли кэцян",
			"всекитайское собрание народных представителей", "политбюро",
			"юань", "китайский юань", "народный банк китая",
			"alibaba", "tencent", "baidu", "huawei", "xiaomi",
			"народно-освободительная армия китая", "нок", "china", "cn",
		},
		"jp": {
			"япония", "японский", "японская", "японское", "японские",
			"токио", "осака", "киото", "иокогама", "нагоя",
			"кисида", "суга", "абе", "коидзуми", "мураками",
			"палата представителей", "палата советников", "правительство японии",
			"йена", "японская йена", "банк японии",
			"toyota", "honda", "nissan", "sony", "panasonic",
			"силы самообороны японии", "япония", "japan", "jp",
		},
		"kz": {
			"казахстан", "казахский", "казахская", "казахское", "казахские",
			"астана", "алматы", "шымкент", "актобе", "тараз",
			"токаев", "назарбаев", "мамин", "атамбаев",
			"мажилис", "сенат", "правительство казахстана",
			"тенге", "казахский тенге", "национальный банк казахстана",
			"казахстан темир жолы", "казмунайгаз", "казпочта",
			"вооруженные силы казахстана", "рк", "kazakhstan", "kz",
		},
		"by": {
			"беларусь", "белорусский", "белорусская", "белорусское", "белорусские",
			"минск", "гомель", "могилев", "витебск", "брест",
			"лукашенко", "головатов", "макеев", "козлов",
			"национальное собрание", "совет республики", "правительство беларуси",
			"белорусский рубль", "национальный банк беларуси",
			"белтрансгаз", "белтелеком", "белгазпромбанк",
			"вооруженные силы беларуси", "рб", "belarus", "by",
		},
	}

	return &CountryDetector{
		logger:   logger,
		keywords: keywords,
	}
}

// DetectCountry определяет страну по контенту новости
func (d *CountryDetector) DetectCountry(title, description, content string) *string {
	// Объединяем весь текст для анализа
	text := strings.ToLower(title + " " + description + " " + content)
	
	// Подсчитываем совпадения ключевых слов для каждой страны
	countryScores := make(map[string]float64)
	
	for countryCode, keywords := range d.keywords {
		score := 0.0
		for _, keyword := range keywords {
			keywordLower := strings.ToLower(keyword)
			if strings.Contains(text, keywordLower) {
				weight := d.calculateCountryKeywordWeight(keyword, text, title, description)
				score += weight
			}
		}
		if score > 0 {
			countryScores[countryCode] = score
		}
	}
	
	// Если нет совпадений, возвращаем nil
	if len(countryScores) == 0 {
		return nil
	}
	
	// Находим страну с максимальным количеством совпадений
	var bestCountry string
	var maxScore float64
	
	for countryCode, score := range countryScores {
		if score > maxScore {
			maxScore = score
			bestCountry = countryCode
		}
	}
	
	// Минимальный порог для уверенности в определении страны
	minThreshold := 0.5
	if maxScore < minThreshold {
		d.logger.WithFields(logrus.Fields{
			"title":         truncateForLog(title, 50),
			"max_score":     maxScore,
			"threshold":     minThreshold,
			"all_scores":    countryScores,
		}).Debug("Country score below threshold, returning nil")
		return nil
	}
	
	// Логируем результат определения страны для отладки
	d.logger.WithFields(logrus.Fields{
		"title":         truncateForLog(title, 50),
		"country_code":  bestCountry,
		"score":         maxScore,
		"all_scores":    countryScores,
	}).Debug("Detected country")
	
	return &bestCountry
}

// calculateCountryKeywordWeight вычисляет вес ключевого слова для определения страны
func (d *CountryDetector) calculateCountryKeywordWeight(keyword, fullText, title, description string) float64 {
	keywordLower := strings.ToLower(keyword)
	weight := 1.0
	
	// Более длинные ключевые слова имеют больший вес
	if len(keyword) > 10 {
		weight *= 1.5
	} else if len(keyword) > 5 {
		weight *= 1.2
	}
	
	// Ключевые слова в заголовке имеют больший вес
	if strings.Contains(strings.ToLower(title), keywordLower) {
		weight *= 2.0
	}
	
	// Ключевые слова в описании имеют средний вес
	if strings.Contains(strings.ToLower(description), keywordLower) {
		weight *= 1.3
	}
	
	// Подсчитываем количество вхождений
	occurrences := strings.Count(fullText, keywordLower)
	if occurrences > 1 {
		weight *= float64(occurrences) * 0.5 // Дополнительный вес за повторения
	}
	
	// Особые веса для важных ключевых слов
	if strings.Contains(keywordLower, "президент") || strings.Contains(keywordLower, "president") {
		weight *= 1.5
	}
	if strings.Contains(keywordLower, "правительство") || strings.Contains(keywordLower, "government") {
		weight *= 1.3
	}
	
	return weight
}

// GetCountryKeywords возвращает ключевые слова для страны
func (d *CountryDetector) GetCountryKeywords(countryCode string) []string {
	if keywords, exists := d.keywords[countryCode]; exists {
		return keywords
	}
	return nil
}

