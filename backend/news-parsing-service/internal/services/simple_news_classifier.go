// internal/classifier/simple_news_classifier.go
package services

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/kljensen/snowball"
	"github.com/sirupsen/logrus"
)

// Конфигурация
type WeightedClassifierConfig struct {
	TitleWeight   float64
	SummaryWeight float64
	ContentWeight float64

	MinConfidence float64
	MinMargin     float64

	UseStemming bool

	CategorySeeds  map[int][]string
	CategoryLabels map[int]string

	URLPriorBoost float64
	BatchTimeout  time.Duration

	AllowUnknown        bool
	MinScoreForFallback float64
	FallbackCategory    int
}

// Классификатор
type WeightedNewsClassifier struct {
	logger *logrus.Logger
	cfg    WeightedClassifierConfig

	posLex    map[int]map[string]float64
	negLex    map[int]map[string]float64
	biLex     map[int]map[string]float64
	stopset   map[string]struct{}
	seedProto map[int]map[string]float64

	stemCache sync.Map

	reToken *regexp.Regexp

	reScore     *regexp.Regexp
	rePercent   *regexp.Regexp
	reCurrency  *regexp.Regexp
	reAI        *regexp.Regexp
	reGov       *regexp.Regexp
	reSociety   *regexp.Regexp
	reTechTerms *regexp.Regexp

	reLaw     *regexp.Regexp
	reBanking *regexp.Regexp
	reAdvice  *regexp.Regexp

	reCrime *regexp.Regexp

	reEdu          *regexp.Regexp
	reQuake        *regexp.Regexp
	reHealth       *regexp.Regexp
	reHealthStrong *regexp.Regexp

	reWar *regexp.Regexp

	// дополнительная явная проверка "выборы/парламент/депутат"
	reExplicitPolitics *regexp.Regexp
}

const (
	CatSport    = 1
	CatTech     = 2
	CatPolitics = 3
	CatEconomy  = 4
	CatSociety  = 5
)

func NewWeightedNewsClassifier(logger *logrus.Logger, cfg WeightedClassifierConfig) (*WeightedNewsClassifier, error) {
	if logger == nil {
		logger = logrus.New()
	}
	// defaults
	if cfg.TitleWeight == 0 {
		cfg.TitleWeight = 1.6
	}
	if cfg.SummaryWeight == 0 {
		cfg.SummaryWeight = 1.0
	}
	if cfg.ContentWeight == 0 {
		cfg.ContentWeight = 2.0
	}
	if cfg.MinConfidence == 0 {
		cfg.MinConfidence = 0.18
	}
	if cfg.MinMargin == 0 {
		cfg.MinMargin = 0.03
	}
	if cfg.BatchTimeout == 0 {
		cfg.BatchTimeout = 30 * time.Second
	}
	if cfg.URLPriorBoost == 0 {
		cfg.URLPriorBoost = 0.30
	}
	if cfg.MinScoreForFallback == 0 {
		cfg.MinScoreForFallback = 0.60
	}
	if cfg.FallbackCategory == 0 {
		cfg.FallbackCategory = CatSociety
	}
	// если сиды не заданы — используем DefaultCategorySeeds
	if cfg.CategorySeeds == nil || len(cfg.CategorySeeds) == 0 {
		cfg.CategorySeeds = DefaultCategorySeeds
	}

	clf := &WeightedNewsClassifier{
		logger:    logger,
		cfg:       cfg,
		posLex:    defaultPositiveLexicon(),
		negLex:    defaultNegativeLexicon(),
		biLex:     defaultBigrams(),
		stopset:   defaultStopwordsRU(),
		seedProto: map[int]map[string]float64{},
		reToken:   regexp.MustCompile(`[\p{L}]+`),

		reScore:     regexp.MustCompile(`\b\d{1,2}\s*[:\-]\s*\d{1,2}\b|со\s+счетом\s+\d+[:\-]\d+`),
		rePercent:   regexp.MustCompile(`\b\d{1,3}([.,]\d{1,2})?\s*%`),
		reCurrency:  regexp.MustCompile(`[$€₽]|руб(ль|лей|\.?)\b|доллар(ов|а)?\b|евро\b`),
		reAI:        regexp.MustCompile(`\b(ии|ai)\b`),
		reGov:       regexp.MustCompile(`(?i)(президент|премьер|министр|депутат|парламент|правительств|санкци|мид|дипломат|саммит|переговор|выбор|выборы|партия|партий|государств|власть|администрац|сенат|конгресс|канц|нато|оон|посольств|войн|всу|фронт|кремл|палестин|признан|дипломатическ|международн|полигон|трамп|сша|польш|прибалтик|эскалац|отношен|росси|варшав|марш|боев|украин|спецпредставител|планы|мэр|санкционн|путин|песков|форум|глобальн|атомн|мирн|неделя|поздравил|оценил|рассказал|ценит|примет\s+участие|приедет|сравнил|заявил|сообщил|конструктивн|отношен|день\s+независимости|академик|передвижн|штаб|антидронов|защит|доменн|печ|увернул|падающ|мотоцикл|иноземцев|особое\s+отношение|пашинян|армени|профсоюз|забастовк|демонстрац|предател|ступников|развод|рахмон|таджикистан|соединенные\s+штаты|америки|эмомали|отбыл)`),
		reSociety:   regexp.MustCompile(`\b(полици|суд|школ|университет|лицей|дет(и|ск)|семь|жители|волонтер|пожар|дтп|авар(и|ия))\w*`),
		reTechTerms: regexp.MustCompile(`\b(смартфон|приложен|стартап|iphone|android|чип|процессор|робот)\w*`),

		reLaw:     regexp.MustCompile(`\b(№\s*\d+\s*-\s*фз|фз\s*№\s*\d+|фз\s*\d+)\b`),
		reBanking: regexp.MustCompile(`\b(банк|банковск|кар(т|точ)|перевод|транзакц|счет|блокировк|финмониторинг|росфинмониторинг)\w*`),
		reAdvice:  regexp.MustCompile(`\b(как\s+действовать|что\s+делать|совет(ы|ует)|рекомендует(ся)?|лучше\s+избегать)\b`),

		reCrime:  regexp.MustCompile(`\b(осужден|осуждён|приговор|приговорен|суд|следственн|уголовн|преступлен|насили|потерпевш|изнасилован|следком)\w*`),
		reEdu:    regexp.MustCompile(`\b(вуз(ы|ов)?|университет|институт|прием\s+в\s+вуз|абитуриент|егэ|бакалавриат|магистратура)\b`),
		reQuake:  regexp.MustCompile(`\b(землетрясен\w*|афтершок\w*|подземн\w+\s+толчк\w*|магнитуд\w*|эпицентр)\b`),
		reHealth: regexp.MustCompile(`\b(аборт\w*|рождаемост\w*|беременност\w*|репродуктивн\w*|здоровь\w*|медицин\w*)\b`),

		reHealthStrong:     regexp.MustCompile(`\b(рак|онколог\w*|опухол\w*|химиотерап\w*|иммунотерап\w*|лекарств\w*|вакцин\w*|пациент|учен(ые|ый|ых))\w*`),
		reWar:              regexp.MustCompile(`(?i)\b(обстрел|удар(ы|ов)?|ракет\w*|дрон\w*|бпла|фронт|всу|вс\s*рф|сбит\w*|арм\w+)\b`),
		reExplicitPolitics: regexp.MustCompile(`(?i)\b(выбор|выборы|депутат|парламент|голосование|референдум|избирател|избир)\b`),
	}

	// если есть сиды - построить прототипы
	if len(cfg.CategorySeeds) > 0 {
		clf.buildSeedPrototypes(cfg.CategorySeeds)
		clf.augmentBigramsFromSeeds(cfg.CategorySeeds)
	}

	return clf, nil
}

func (c *WeightedNewsClassifier) Close() error { return nil }

func (c *WeightedNewsClassifier) ProcessNewsBatch(ctx context.Context, items []UnifiedNewsItem) ([]UnifiedProcessingResult, error) {
	ctx, cancel := context.WithTimeout(ctx, c.cfg.BatchTimeout)
	defer cancel()

	results := make([]UnifiedProcessingResult, len(items))
	for i, it := range items {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}
		results[i] = c.classify(it, it.Index)
	}
	return results, nil
}

func (c *WeightedNewsClassifier) classify(item UnifiedNewsItem, idx int) UnifiedProcessingResult {
	if strings.TrimSpace(item.Title) == "" {
		c.logger.WithField("index", idx).Warn("Empty title in news item")
		return UnifiedProcessingResult{Index: idx, Title: item.Title, Content: item.Content, CategoryID: 0, Confidence: 0.0, Error: fmt.Errorf("empty title")}
	}

	title := c.normalize(item.Title)
	desc := c.normalize(item.Description)
	body := c.normalize(item.Content)
	if body == "" {
		body = desc
	}

	// Токены/биграммы
	tTitle := c.tokens(title)
	tDesc := c.tokens(desc)
	tBody := c.tokens(body)
	biTitle := bigrams(tTitle)
	biDesc := bigrams(tDesc)
	biBody := bigrams(tBody)

	// Вектор документа
	doc := map[string]float64{}
	addTokens(doc, tTitle, c.cfg.TitleWeight)
	addTokens(doc, tDesc, c.cfg.SummaryWeight)
	addTokens(doc, tBody, c.cfg.ContentWeight)
	addTokens(doc, biTitle, 1.8*c.cfg.TitleWeight)
	addTokens(doc, biDesc, 1.4*c.cfg.SummaryWeight)
	addTokens(doc, biBody, 2.0*c.cfg.ContentWeight)

	rawAll := title + " " + desc + " " + body

	hasGov := c.reGov.MatchString(rawAll) || c.reWar.MatchString(rawAll)
	hasSportCore := strings.Contains(rawAll, "матч") || strings.Contains(rawAll, "чемпионат") ||
		strings.Contains(rawAll, "турнир") || strings.Contains(rawAll, "лига") ||
		strings.Contains(rawAll, "клуб") || strings.Contains(rawAll, "команд") ||
		strings.Contains(rawAll, "гол") || strings.Contains(rawAll, "спорт") ||
		strings.Contains(rawAll, "фигурн") || strings.Contains(rawAll, "катани") ||
		strings.Contains(rawAll, "олимпиад") || strings.Contains(rawAll, "игр") ||
		strings.Contains(rawAll, "формул") || strings.Contains(rawAll, "гонк") ||
		strings.Contains(rawAll, "гран при") || strings.Contains(rawAll, "полуфинал") ||
		strings.Contains(rawAll, "четвертьфинал") || strings.Contains(rawAll, "теннис") ||
		strings.Contains(rawAll, "гуменник") || strings.Contains(rawAll, "балтика") ||
		strings.Contains(rawAll, "рпл") || strings.Contains(rawAll, "триумф") ||
		strings.Contains(rawAll, "порадовал") || strings.Contains(rawAll, "вдохновил") ||
		strings.Contains(rawAll, "удивил") || strings.Contains(rawAll, "поедет") ||
		strings.Contains(rawAll, "выступление") || strings.Contains(rawAll, "олимпийский") ||
		strings.Contains(rawAll, "фигурист") || strings.Contains(rawAll, "фигуристка") ||
		strings.Contains(rawAll, "прокат") || strings.Contains(rawAll, "балл") ||
		strings.Contains(rawAll, "оценк") || strings.Contains(rawAll, "медал") ||
		strings.Contains(rawAll, "прыжок") || strings.Contains(rawAll, "прыжки") ||
		strings.Contains(rawAll, "вращени") || strings.Contains(rawAll, "каскад") ||
		strings.Contains(rawAll, "комбинац") || strings.Contains(rawAll, "спирал") ||
		strings.Contains(rawAll, "аксель") || strings.Contains(rawAll, "лутц") ||
		strings.Contains(rawAll, "флип") || strings.Contains(rawAll, "сальхов") ||
		strings.Contains(rawAll, "риттбергер") || strings.Contains(rawAll, "тулуп") ||
		strings.Contains(rawAll, "квот") || strings.Contains(rawAll, "путевк") ||
		strings.Contains(rawAll, "пробилс") || strings.Contains(rawAll, "завоевал") ||
		strings.Contains(rawAll, "isu") || strings.Contains(rawAll, "конькобежн") ||
		strings.Contains(rawAll, "союз") || strings.Contains(rawAll, "международн") ||
		strings.Contains(rawAll, "зимн") || strings.Contains(rawAll, "милан") ||
		strings.Contains(rawAll, "кортин") || strings.Contains(rawAll, "итали") ||
		strings.Contains(rawAll, "юниорск") || strings.Contains(rawAll, "взросл") ||
		strings.Contains(rawAll, "европ") || strings.Contains(rawAll, "гран") ||
		strings.Contains(rawAll, "при") || strings.Contains(rawAll, "этап") ||
		strings.Contains(rawAll, "финал") || strings.Contains(rawAll, "кубок") ||
		strings.Contains(rawAll, "российск") || strings.Contains(rawAll, "сборн") ||
		strings.Contains(rawAll, "олимпийск") || strings.Contains(rawAll, "программа") ||
		strings.Contains(rawAll, "хореографи") || strings.Contains(rawAll, "судь") ||
		strings.Contains(rawAll, "техническ") || strings.Contains(rawAll, "элемент") ||
		strings.Contains(rawAll, "сил") || strings.Contains(rawAll, "страст") ||
		strings.Contains(rawAll, "точност")

	hasEconCore := c.reCurrency.MatchString(rawAll) || strings.Contains(rawAll, "инфляц") ||
		strings.Contains(rawAll, "ставк") || strings.Contains(rawAll, "рынк") ||
		strings.Contains(rawAll, "бюджет") || strings.Contains(rawAll, "акци")

	type parts struct{ lex, seed, pat, prior float64 }
	type cand struct {
		cat   int
		score float64
		parts parts
	}
	var cands []cand

	for cat := range c.posLex {
		lex := dotWithLex(doc, c.posLex[cat]) - dotWithLex(doc, c.negLex[cat])
		seed := cosineWithProto(doc, c.seedProto[cat])
		pat := c.patternBoost(cat, rawAll)
		prior := c.urlPrior(cat, item.URL)

		score := 1.2*lex + 3.0*seed + 1.4*pat + prior

		switch cat {
		case CatSport:
			if hasSportCore || c.reScore.MatchString(rawAll) {
				score += 20.0 // Еще больше увеличили бонус за спортивные термины
			} else {
				score -= 1.4
			}
			if hasGov {
				score -= 25.0 // Увеличили штраф с 15.0 до 25.0
			}
			if c.reHealthStrong.MatchString(rawAll) {
				score -= 3.0
			}
		case CatEconomy:
			if !hasEconCore {
				score -= 0.8
			}
		case CatPolitics:
			if hasGov {
				score += 5.0 // Увеличили boost с 3.0 до 5.0
			}
		}

		cands = append(cands, cand{cat: cat, score: score, parts: parts{lex: lex, seed: seed, pat: pat, prior: prior}})
	}

	if len(cands) == 0 {
		return UnifiedProcessingResult{Index: idx, Title: item.Title, Content: body, CategoryID: 0, Confidence: 0.0, Error: fmt.Errorf("no candidates")}
	}

	sort.Slice(cands, func(i, j int) bool { return cands[i].score > cands[j].score })
	best := cands[0]
	var second cand
	if len(cands) > 1 {
		second = cands[1]
	}

	confProto := clamp01(best.parts.seed)
	margin := (best.score - second.score) / (math.Abs(best.score) + math.Abs(second.score) + 1e-9)
	if margin < 0 {
		margin = 0
	}
	conf := clamp01(0.7*confProto + 0.3*margin)

	category := best.cat
	var err error

	lowConf := conf < c.cfg.MinConfidence || margin < c.cfg.MinMargin
	if lowConf {
		if best.score >= c.cfg.MinScoreForFallback {
			err = fmt.Errorf("low confidence but applied fallback: conf=%.3f margin=%.3f best=%d score=%.3f", conf, margin, best.cat, best.score)
			conf = clamp01(0.5*conf + 0.5*sigmoid(best.score))
		} else if !c.cfg.AllowUnknown {
			category = c.cfg.FallbackCategory
			err = fmt.Errorf("assigned fallback category due to low confidence: conf=%.3f margin=%.3f best=%d score=%.3f -> fallback=%d", conf, margin, best.cat, best.score, category)
			if conf < 0.25 {
				conf = 0.25
			}
		} else {
			category = 0
			err = fmt.Errorf("low confidence: conf=%.3f margin=%.3f best=%d score=%.3f", conf, margin, best.cat, best.score)
		}
	}

	// демотирование спорта при сильных мед.терминах
	if category == CatSport && c.reHealthStrong.MatchString(rawAll) {
		category = CatSociety
		conf = max(conf, 0.7)
		err = fmt.Errorf("demoted sport -> society due to health/medical terms")
	}

	content := item.Content
	if content == "" {
		content = item.Description
	}

	// подробный лог
	if c.logger.IsLevelEnabled(logrus.DebugLevel) || hasGov || category == CatPolitics || category == CatSport {
		c.logger.WithFields(logrus.Fields{
			"idx":     idx,
			"title":   item.Title,
			"best":    category,
			"conf":    fmt.Sprintf("%.3f", conf),
			"margin":  fmt.Sprintf("%.3f", margin),
			"seed":    fmt.Sprintf("%.3f", best.parts.seed),
			"lex":     fmt.Sprintf("%.3f", best.parts.lex),
			"pat":     fmt.Sprintf("%.2f", best.parts.pat),
			"prior":   fmt.Sprintf("%.2f", best.parts.prior),
			"second":  second.cat,
			"score_b": fmt.Sprintf("%.3f", best.score),
			"score_s": fmt.Sprintf("%.3f", second.score),
			"has_gov": hasGov,
		}).Info("classification detail")
	}

	return UnifiedProcessingResult{
		Index:      idx,
		Title:      item.Title,
		Content:    content,
		CategoryID: category,
		Confidence: round(conf, 3),
		Error:      err,
	}
}

// isExplicitPolitical — жёсткая проверка по заголовку/короткому контенту.
// Возвращает true если в заголовке/коротком описании встречаются явные маркеры выборов/депутатов/парламента.
func (c *WeightedNewsClassifier) isExplicitPolitical(title, desc, body string) bool {
	if c.reExplicitPolitics.MatchString(title) {
		return true
	}
	// если упоминается "депутат" или "парламент" рядом с названием страны -> очень явно
	if c.reExplicitPolitics.MatchString(desc) {
		return true
	}
	if c.reExplicitPolitics.MatchString(body) {
		return true
	}
	return false
}

//
// patternBoost, urlPrior, utilities, TF-IDF prototyping и словари — те же, что раньше
// (скопированы/сохранены для полноты и не изменены логикой, кроме более строгой политики выше).
//

func (c *WeightedNewsClassifier) patternBoost(cat int, raw string) float64 {
	switch cat {
	case CatSport:
		boost := 0.0
		if c.reGov.MatchString(raw) || c.reWar.MatchString(raw) {
			return -2.0
		}
		if c.reHealthStrong.MatchString(raw) {
			boost -= 1.0
		}
		if c.reScore.MatchString(raw) {
			if strings.Contains(raw, "матч") || strings.Contains(raw, "игра") ||
				strings.Contains(raw, "чемпионат") || strings.Contains(raw, "турнир") ||
				strings.Contains(raw, "лига") || strings.Contains(raw, "клуб") ||
				strings.Contains(raw, "команд") || strings.Contains(raw, "гол") ||
				strings.Contains(raw, "формул") || strings.Contains(raw, "гонк") ||
				(strings.Contains(raw, "гран") && strings.Contains(raw, "при")) {
				boost += 0.9
			}
		}
		if strings.Contains(raw, "тур ") || strings.Contains(raw, "турнир") || strings.Contains(raw, "финал") {
			boost += 0.4
		}
		if strings.Contains(raw, "гол") || strings.Contains(raw, "голы") {
			boost += 0.3
		}
		if c.reCrime.MatchString(raw) {
			boost -= 0.6
			if boost < 0 {
				boost = 0
			}
		}
		return boost

	case CatEconomy:
		boost := 0.0
		if c.reCurrency.MatchString(raw) {
			boost += 0.6
		}
		if c.rePercent.MatchString(raw) || strings.Contains(raw, "ставк") ||
			strings.Contains(raw, "ввп") || strings.Contains(raw, "инфляц") {
			boost += 0.5
		}
		if c.reBanking.MatchString(raw) && c.reLaw.MatchString(raw) && !c.reGov.MatchString(raw) {
			boost += 0.7
		}
		if c.reEdu.MatchString(raw) || c.reHealth.MatchString(raw) || c.reQuake.MatchString(raw) {
			boost -= 0.6
			if boost < 0 {
				boost = 0
			}
		}
		return boost

	case CatPolitics:
		boost := 0.0
		if c.reGov.MatchString(raw) || c.reWar.MatchString(raw) {
			boost += 0.6
		}
		return boost

	case CatTech:
		boost := 0.0
		if c.reAI.MatchString(raw) {
			boost += 0.6
		}
		if c.reTechTerms.MatchString(raw) {
			boost += 0.5
		}
		return boost

	case CatSociety:
		boost := 0.0
		if c.reSociety.MatchString(raw) {
			boost += 0.5
		}
		if c.reBanking.MatchString(raw) && c.reAdvice.MatchString(raw) {
			boost += 0.6
		}
		if c.reLaw.MatchString(raw) && c.reAdvice.MatchString(raw) {
			boost += 0.4
		}
		if c.reCrime.MatchString(raw) {
			boost += 0.7
		}
		if c.reEdu.MatchString(raw) {
			boost += 0.7
		}
		if c.reQuake.MatchString(raw) {
			boost += 0.8
		}
		if c.reHealth.MatchString(raw) {
			boost += 0.7
		}
		if c.reHealthStrong.MatchString(raw) {
			boost += 0.8
		}
		return boost
	default:
		return 0.0
	}
}

func (c *WeightedNewsClassifier) urlPrior(cat int, url string) float64 {
	if url == "" {
		return 0
	}
	u := strings.ToLower(url)
	b := c.cfg.URLPriorBoost
	switch cat {
	case CatSport:
		if strings.Contains(u, "/sport") || strings.Contains(u, "/sports") {
			return b
		}
	case CatTech:
		if strings.Contains(u, "/tech") || strings.Contains(u, "/hi-tech") || strings.Contains(u, "/it") {
			return b
		}
	case CatPolitics:
		if strings.Contains(u, "/polit") || strings.Contains(u, "/politics") || strings.Contains(u, "/world") {
			return b
		}
	case CatEconomy:
		if strings.Contains(u, "/econom") || strings.Contains(u, "/business") || strings.Contains(u, "/finance") {
			return b
		}
	case CatSociety:
		if strings.Contains(u, "/society") || strings.Contains(u, "/life") || strings.Contains(u, "/city") {
			return b
		}
	}
	return 0
}

func (c *WeightedNewsClassifier) normalize(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "ё", "е")
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.TrimSpace(s)
}

func (c *WeightedNewsClassifier) tokens(s string) []string {
	if s == "" {
		return nil
	}
	raw := c.reToken.FindAllString(s, -1)
	if len(raw) == 0 {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, t := range raw {
		if _, stop := c.stopset[t]; stop {
			continue
		}
		if c.cfg.UseStemming {
			if stemmed, ok := c.stemCache.Load(t); ok {
				t = stemmed.(string)
			} else {
				stem, err := snowball.Stem(t, "russian", true)
				if err == nil && stem != "" {
					c.stemCache.Store(t, stem)
					t = stem
				}
			}
		}
		if len(t) <= 2 {
			continue
		}
		out = append(out, t)
	}
	return out
}

func bigrams(tokens []string) []string {
	if len(tokens) < 2 {
		return nil
	}
	out := make([]string, 0, len(tokens)-1)
	for i := 0; i+1 < len(tokens); i++ {
		out = append(out, tokens[i]+" "+tokens[i+1])
	}
	return out
}

func addTokens(dst map[string]float64, toks []string, w float64) {
	for _, t := range toks {
		dst[t] += w
	}
}

func dotWithLex(doc map[string]float64, lex map[string]float64) float64 {
	if len(lex) == 0 || len(doc) == 0 {
		return 0
	}
	var s float64
	for t, w := range doc {
		if lw, ok := lex[t]; ok {
			s += w * lw
		}
	}
	return s
}

func cosineWithProto(doc map[string]float64, proto map[string]float64) float64 {
	if len(proto) == 0 || len(doc) == 0 {
		return 0
	}
	var dot, na, nb float64
	for t, w := range doc {
		if pw, ok := proto[t]; ok {
			dot += w * pw
		}
		na += w * w
	}
	for _, pw := range proto {
		nb += pw * pw
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

func round(x float64, prec int) float64 {
	p := math.Pow(10, float64(prec))
	return math.Round(x*p) / p
}

func sigmoid(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}

func (c *WeightedNewsClassifier) buildSeedPrototypes(seeds map[int][]string) {
	type doc struct {
		cat int
		tf  map[string]float64
	}
	var docs []doc
	df := map[string]int{}

	for cat, list := range seeds {
		for _, s := range list {
			toks := c.tokens(c.normalize(s))
			if len(toks) == 0 {
				continue
			}
			tf := map[string]float64{}
			seen := map[string]struct{}{}
			for _, t := range toks {
				tf[t]++
			}
			for t := range tf {
				if _, ok := seen[t]; !ok {
					df[t]++
					seen[t] = struct{}{}
				}
			}
			docs = append(docs, doc{cat: cat, tf: tf})
		}
	}

	if len(docs) == 0 {
		return
	}
	N := float64(len(docs))

	byCat := map[int][]map[string]float64{}
	for _, d := range docs {
		tfidf := map[string]float64{}
		var maxTF float64
		for _, tf := range d.tf {
			if tf > maxTF {
				maxTF = tf
			}
		}
		for t, tf := range d.tf {
			idf := math.Log((N+1.0)/float64(df[t]) + 1.0)
			tfidf[t] = (tf / (maxTF + 1e-9)) * idf
		}
		byCat[d.cat] = append(byCat[d.cat], tfidf)
	}

	c.seedProto = map[int]map[string]float64{}
	for cat, vecs := range byCat {
		proto := map[string]float64{}
		for _, v := range vecs {
			for t, w := range v {
				proto[t] += w
			}
		}
		var norm float64
		for _, w := range proto {
			norm += w * w
		}
		norm = math.Sqrt(norm)
		if norm > 0 {
			for t, w := range proto {
				proto[t] = w / norm
			}
		}
		c.seedProto[cat] = proto
	}
}

func (c *WeightedNewsClassifier) augmentBigramsFromSeeds(seeds map[int][]string) {
	for cat, list := range seeds {
		if _, ok := c.biLex[cat]; !ok {
			c.biLex[cat] = map[string]float64{}
		}
		for _, s := range list {
			toks := c.tokens(c.normalize(s))
			for _, bg := range bigrams(toks) {
				c.biLex[cat][bg] += 1.2
			}
		}
	}
}

func defaultPositiveLexicon() map[int]map[string]float64 {
	return map[int]map[string]float64{
		CatSport: {
			"спорт": 0.8, "матч": 0.8, "команд": 0.7, "игрок": 0.7, "тренер": 0.6, "чемпионат": 0.8,
			"турнир": 0.7, "гол": 0.8, "счет": 0.6, "лига": 0.6, "клуб": 0.7, "футбол": 0.9,
			"хоккей": 0.8, "баскетбол": 0.7, "теннис": 0.6, "олимпиад": 1.0,
			"фигурн": 1.2, "катани": 1.2, "прокат": 1.0, "балл": 0.9, "оценк": 0.9, "медал": 0.9, "победител": 0.8,
			"формул": 1.2, "болид": 1.0, "гонк": 1.1, "пилот": 0.9, "трасс": 0.8, "квалификац": 0.8,
			"полуфинал": 1.0, "четвертьфинал": 1.0,
			"гуменник": 1.2, "балтика": 1.1, "рпл": 1.0, "триумф": 0.8, "порадовал": 0.6, "вдохновил": 0.6,
			"удивил": 0.6, "поедет": 0.7, "поедем": 0.7, "игры": 0.9, "выступление": 0.9, "олимпийский": 1.1,
			"фигурист": 1.3, "фигуристка": 1.3, "программа": 0.8, "хореографи": 0.7, "судь": 0.6,
			"техническ": 0.7, "элемент": 0.7, "прыжок": 0.8, "прыжки": 0.8, "вращени": 0.7,
			"каскад": 0.8, "комбинац": 0.7, "спирал": 0.7, "аксель": 0.9, "лутц": 0.8, "флип": 0.8,
			"сальхов": 0.8, "риттбергер": 0.8, "тулуп": 0.8, "квот": 0.8, "путевк": 0.8,
			"пробилс": 0.8, "завоевал": 0.8, "сил": 0.6, "страст": 0.6, "точност": 0.6,
			"isu": 1.0, "конькобежн": 0.9, "союз": 0.6, "международн": 0.6, "зимн": 0.8,
			"милан": 0.7, "кортин": 0.7, "итали": 0.6, "юниорск": 0.7, "взросл": 0.6,
			"европ": 0.7, "гран": 0.8, "при": 0.6, "этап": 0.7, "финал": 0.8, "кубок": 0.8,
			"российск": 0.6, "сборн": 0.7, "олимпийск": 1.0,
		},
		CatTech: {
			"технолог": 1.5, "компьютер": 1.2, "интернет": 1.2, "смартфон": 1.2, "приложен": 1.2,
			"искусственн": 1.6, "интеллект": 1.6, "робот": 1.2, "софт": 1.1, "данн": 1.1, "сет": 1.1,
			"гаджет": 1.3, "стартап": 1.4, "инновац": 1.4, "программ": 1.4,
		},
		CatPolitics: {
			"президент": 2.0, "правительств": 2.0, "выбор": 2.0, "парламент": 1.8, "министр": 1.8,
			"депутат": 1.8, "закон": 1.2,
			"санкци": 1.8, "переговор": 1.6, "дипломат": 1.6,
			"войн": 1.6, "безопасн": 1.4, "реформ": 1.4, "кремл": 2.0, "нато": 1.6, "ес": 1.4, "оон": 1.4,
		},
		CatEconomy: {
			"эконом": 1.6, "инфляц": 1.4, "валют": 1.3, "курс": 1.2, "рубл": 1.2, "доллар": 1.2, "евро": 1.2,
			"бюджет": 1.3, "банк": 1.2, "кредит": 1.1, "инвестиц": 1.3, "рынк": 1.2, "налог": 1.2, "ставк": 1.3,
			"финанс": 1.4, "бизнес": 1.4, "компан": 1.4, "прибыл": 1.2, "акци": 1.2, "ввп": 1.2,
		},
		CatSociety: {
			"обществен": 1.3, "социальн": 1.3, "граждан": 1.1, "суд": 1.0, "полици": 1.0, "дорог": 1.0,
			"семь": 1.0, "дет": 1.0, "образован": 1.2, "школ": 1.2, "университет": 1.1, "транспорт": 1.1,
			"пожар": 1.2, "дтп": 1.2, "авари": 1.1, "волонтер": 1.0, "жители": 1.0, "инициатив": 1.0,
			"культур": 1.4, "искусств": 1.3, "театр": 1.3, "кино": 1.3, "фильм": 1.2,
			"здоровь": 1.0, "медицин": 1.0, "учен": 0.9, "исследован": 0.9, "пациент": 1.0, "врач": 1.0,
		},
	}
}

func defaultNegativeLexicon() map[int]map[string]float64 {
	return map[int]map[string]float64{
		CatSport: {
			"политик": -12.0, "президент": -12.0, "правительств": -12.0, "выбор": -12.0, "парламент": -12.0,
			"министр": -12.0, "депутат": -12.0, "закон": -10.0, "санкци": -12.0,
			"вуз": -6.0, "егэ": -6.0, "землетрясен": -6.0, "пожар": -6.0,
			"рак": -10.0, "онколог": -10.0,
		},
		CatEconomy: {
			"матч": -2.0, "гол": -2.0, "турнир": -2.0, "чемпионат": -2.0, "спорт": -2.0,
			"аборт": -2.0, "рождаемост": -2.0,
		},
		CatPolitics: {
			"матч": -2.0, "гол": -2.0, "лига": -2.0, "спорт": -2.0,
			"банк": 0.6, "карта": 0.6, "перевод": 0.6,
		},
		CatTech: {
			"инфляц": -1.5, "ставк": -1.5, "курс": -1.5,
			"спорт": -1.5, "матч": -1.5,
		},
		CatSociety: {
			"матч": -3.0, "гол": -3.0, "турнир": -3.0, "чемпионат": -3.0, "спорт": -3.0,
		},
	}
}

func defaultBigrams() map[int]map[string]float64 {
	return map[int]map[string]float64{
		CatSport:    {"лига чемпионов": 2.0, "чемпионат мира": 1.9, "гран при": 2.0, "формула 1": 2.2},
		CatTech:     {"искусственный интеллект": 2.2, "машинное обучение": 1.8, "большие данные": 1.6},
		CatPolitics: {"совет безопасности": 1.8, "главы государств": 1.7},
		CatEconomy:  {"ключевая ставка": 1.8, "валютный рынок": 1.6, "фондовый рынок": 1.6},
	}
}

func defaultStopwordsRU() map[string]struct{} {
	words := []string{
		"и", "в", "во", "не", "на", "но", "что", "он", "она", "они", "мы", "вы",
		"к", "ко", "от", "до", "за", "из", "у", "по", "со", "ли", "а", "же", "как",
		"так", "для", "это", "то", "также", "при", "о", "об", "над", "под", "с",
		"бы", "был", "была", "были", "быть", "или",
	}
	m := make(map[string]struct{}, len(words))
	for _, w := range words {
		m[w] = struct{}{}
	}
	return m
}

// SimpleNewsClassifier alias
type SimpleNewsClassifier = WeightedNewsClassifier

func NewSimpleNewsClassifier(logger *logrus.Logger) *SimpleNewsClassifier {
	cfg := WeightedClassifierConfig{
		TitleWeight:   1.6,
		SummaryWeight: 1.0,
		ContentWeight: 1.0,

		MinConfidence: 0.18,
		MinMargin:     0.03,

		UseStemming:   true,
		URLPriorBoost: 0.30,
		BatchTimeout:  30 * time.Second,

		AllowUnknown:        false,
		MinScoreForFallback: 0.60,
		FallbackCategory:    CatSociety,
	}

	cfg.CategorySeeds = DefaultCategorySeeds

	clf, err := NewWeightedNewsClassifier(logger, cfg)
	if err != nil {
		logger.WithError(err).Error("Failed to create weighted classifier, fallback to simple")
		return &WeightedNewsClassifier{logger: logger, cfg: cfg}
	}
	return clf
}

// Типы
type UnifiedNewsItem struct {
	Index       int
	Title       string
	Description string
	Content     string
	URL         string
	Categories  []string
}

type UnifiedProcessingResult struct {
	Index      int
	Title      string
	Content    string
	CategoryID int
	Confidence float64
	Error      error
}
