package services

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
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

	CategorySeeds  map[int][]string // кат -> набор коротких сид-текстов
	CategoryLabels map[int]string

	BatchTimeout time.Duration
}

// Классификатор
type WeightedNewsClassifier struct {
	logger *logrus.Logger
	cfg    WeightedClassifierConfig

	posLex    map[int]map[string]float64 // cat -> token -> weight
	negLex    map[int]map[string]float64 // cat -> token -> weight
	biLex     map[int]map[string]float64 // cat -> "w1 w2" -> weight
	stopset   map[string]struct{}
	seedProto map[int]map[string]float64 // TF-IDF прототипы

	reToken *regexp.Regexp
}

func NewWeightedNewsClassifier(logger *logrus.Logger, cfg WeightedClassifierConfig) (*WeightedNewsClassifier, error) {
	if logger == nil {
		logger = logrus.New()
	}
	if cfg.TitleWeight == 0 {
		cfg.TitleWeight = 1.6
	}
	if cfg.SummaryWeight == 0 {
		cfg.SummaryWeight = 1.0
	}
	if cfg.ContentWeight == 0 {
		cfg.ContentWeight = 1.0
	}
	if cfg.MinConfidence == 0 {
		cfg.MinConfidence = 0.1
	}
	if cfg.MinMargin == 0 {
		cfg.MinMargin = 0.10
	}
	if cfg.BatchTimeout == 0 {
		cfg.BatchTimeout = 30 * time.Second
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
	}

	if len(cfg.CategorySeeds) > 0 {
		clf.buildSeedPrototypes(cfg.CategorySeeds)
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
		results[i] = c.classify(it, i+1)
	}
	return results, nil
}

func (c *WeightedNewsClassifier) classify(item UnifiedNewsItem, idx int) UnifiedProcessingResult {
	// Валидация входных данных
	if item.Title == "" {
		c.logger.WithField("index", idx).Warn("Empty title in news item")
		return UnifiedProcessingResult{Index: idx, Title: item.Title, Content: item.Content, CategoryID: 1, Confidence: 0.1, Error: nil}
	}

	title := c.normalize(item.Title)
	desc := c.normalize(item.Description)
	body := c.normalize(item.Content)
	if body == "" {
		body = desc
	}

	tTitle := c.tokens(title)
	tDesc := c.tokens(desc)
	tBody := c.tokens(body)
	biTitle := bigrams(tTitle)
	biDesc := bigrams(tDesc)
	biBody := bigrams(tBody)

	doc := map[string]float64{}
	addTokens(doc, tTitle, c.cfg.TitleWeight)
	addTokens(doc, tDesc, c.cfg.SummaryWeight)
	addTokens(doc, tBody, c.cfg.ContentWeight)
	addTokens(doc, biTitle, 1.8*c.cfg.TitleWeight)
	addTokens(doc, biDesc, 1.4*c.cfg.SummaryWeight)
	addTokens(doc, biBody, 1.2*c.cfg.ContentWeight)

	type cand struct {
		cat   int
		score float64
		seed  float64
	}
	var cands []cand
	for cat := range c.posLex {
		pos := dotWithLex(doc, c.posLex[cat]) - dotWithLex(doc, c.negLex[cat])
		seed := cosineWithProto(doc, c.seedProto[cat]) // 0..1
		score := pos + 2.0*seed
		cands = append(cands, cand{cat: cat, score: score, seed: seed})
	}

	if len(cands) == 0 {
		return UnifiedProcessingResult{Index: idx, Title: item.Title, Content: body, CategoryID: 1, Confidence: 0.1, Error: nil} // По умолчанию - Политика
	}

	sort.Slice(cands, func(i, j int) bool { return cands[i].score > cands[j].score })
	best := cands[0]
	var second cand
	if len(cands) > 1 {
		second = cands[1]
	}

	confProto := clamp01(best.seed) // 0..1
	margin := 0.0
	if len(cands) > 1 {
		margin = (best.score - second.score) / (math.Abs(best.score) + math.Abs(second.score) + 1e-9)
		if margin < 0 {
			margin = 0
		}
	}
	conf := clamp01(0.7*confProto + 0.3*margin)

	category := best.cat
	var err error

	// Если уверенность слишком низкая, считаем классификацию неудачной
	if conf < c.cfg.MinConfidence || margin < c.cfg.MinMargin {
		category = 0 // Индикатор неудачи
		err = fmt.Errorf("classification confidence too low: conf=%.3f, margin=%.3f, min_conf=%.3f, min_margin=%.3f",
			conf, margin, c.cfg.MinConfidence, c.cfg.MinMargin)
	}

	content := item.Content
	if content == "" {
		content = item.Description
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

//
// Текст/векторы
//

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
			if stem, err := snowball.Stem(t, "russian", true); err == nil && stem != "" {
				t = stem
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

//
// Seed TF-IDF прототипы
//

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

//
// Словари по умолчанию
//

func defaultPositiveLexicon() map[int]map[string]float64 {
	return map[int]map[string]float64{
		1: {"президент": 1.6, "правительств": 1.6, "выбор": 1.6, "парламент": 1.5, "министр": 1.4,
			"депутат": 1.4, "закон": 1.4, "санкци": 1.4, "переговор": 1.3, "дипломат": 1.2,
			"войн": 1.2, "безопасн": 1.1, "реформ": 1.1, "кремл": 1.6, "нато": 1.1, "ес": 1.0},
		2: {"эконом": 1.6, "инфляц": 1.4, "валют": 1.3, "курс": 1.2, "рубл": 1.2, "доллар": 1.2, "евро": 1.2,
			"бюджет": 1.3, "банк": 1.2, "кредит": 1.1, "инвестиц": 1.3, "рынк": 1.2, "налог": 1.2, "ставк": 1.3},
		3: {"спорт": 1.5, "матч": 1.5, "команд": 1.3, "игрок": 1.3, "тренер": 1.2, "чемпионат": 1.4,
			"турнир": 1.3, "гол": 1.4, "счет": 1.2, "лига": 1.2, "клуб": 1.3},
		4: {"технолог": 1.5, "компьютер": 1.2, "интернет": 1.2, "смартфон": 1.2, "приложен": 1.2,
			"искусственн": 1.6, "интеллект": 1.6, "робот": 1.2, "софт": 1.1, "данн": 1.1, "сет": 1.1},
		5: {"культур": 1.5, "искусств": 1.4, "музе": 1.2, "театр": 1.2, "кино": 1.2, "фильм": 1.3,
			"актер": 1.2, "режиссер": 1.2, "музык": 1.2, "концерт": 1.2, "выставк": 1.2, "книг": 1.2},
		6: {"наук": 1.6, "исследован": 1.4, "учен": 1.3, "эксперимент": 1.2, "лаборатор": 1.2,
			"журнал": 1.1, "университет": 1.1, "открыт": 1.3, "разработк": 1.2},
		7: {"обществен": 1.3, "социальн": 1.3, "граждан": 1.1, "суд": 1.0, "полици": 1.0,
			"дорог": 1.0, "семь": 1.0, "дет": 1.0},
		8: {"происшеств": 1.5, "дтп": 1.5, "авари": 1.4, "пожар": 1.4, "взрыв": 1.4, "катастроф": 1.4,
			"теракт": 1.5, "криминал": 1.3, "убийств": 1.3, "краж": 1.2, "землетрясен": 1.3},
		9: {"здоров": 1.5, "медицин": 1.4, "врач": 1.3, "больниц": 1.2, "лечен": 1.2, "заболеван": 1.3,
			"симптом": 1.1, "диагноз": 1.1, "вакцин": 1.2, "эпидеми": 1.2},
		10: {"школ": 1.3, "университет": 1.3, "институт": 1.2, "экзамен": 1.2, "студент": 1.2,
			"учител": 1.1, "курс": 1.1, "лекц": 1.1, "бакалавр": 1.1, "магистр": 1.1},
		11: {"международ": 1.4, "саммит": 1.3, "дипломат": 1.2, "посол": 1.1, "виза": 1.1, "границ": 1.1,
			"нато": 1.2, "ес": 1.1, "оон": 1.1, "брюссел": 1.0},
		12: {"бизнес": 1.4, "компан": 1.4, "корпорац": 1.3, "директор": 1.2, "менеджер": 1.1,
			"сотрудник": 1.1, "офис": 1.0, "рекрутинг": 1.0, "уволен": 1.1, "прибыл": 1.2},
	}
}

func defaultNegativeLexicon() map[int]map[string]float64 {
	return map[int]map[string]float64{
		3: {"инфляц": 0.4, "ставк": 0.4, "курс": 0.4, "врач": 0.4, "лечен": 0.4},
		2: {"матч": 0.6, "гол": 0.6, "турнир": 0.5, "чемпионат": 0.5},
		1: {"матч": 0.4, "гол": 0.4, "турнир": 0.3, "лига": 0.3},
	}
}

func defaultBigrams() map[int]map[string]float64 {
	return map[int]map[string]float64{
		4: {"искусственный интеллект": 2.2, "машинное обучение": 1.8, "большие данные": 1.6},
		3: {"лига чемпионов": 2.0, "чемпионат мира": 1.9, "мировой рекорд": 1.6},
		1: {"совет безопасности": 1.8, "главы государств": 1.7, "мирный договор": 1.5},
		2: {"ключевая ставка": 1.8, "валютный рынок": 1.6, "фондовый рынок": 1.6},
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

// SimpleNewsClassifier - алиас для обратной совместимости
type SimpleNewsClassifier = WeightedNewsClassifier

// NewSimpleNewsClassifier создает новый простой классификатор (обратная совместимость)
func NewSimpleNewsClassifier(logger *logrus.Logger) *SimpleNewsClassifier {
	cfg := WeightedClassifierConfig{
		TitleWeight:   1.6,
		SummaryWeight: 1.0,
		ContentWeight: 1.0,
		MinConfidence: 0.1,
		MinMargin:     0.05,
		UseStemming:   true,
		BatchTimeout:  30 * time.Second,
	}

	clf, err := NewWeightedNewsClassifier(logger, cfg)
	if err != nil {
		logger.WithError(err).Error("Failed to create weighted classifier, falling back to simple")
		// Fallback к простому классификатору если что-то пошло не так
		return &WeightedNewsClassifier{
			logger: logger,
			cfg:    cfg,
		}
	}
	return clf
}

// UnifiedNewsItem представляет элемент новости для объединенной обработки
type UnifiedNewsItem struct {
	Index       int
	Title       string
	Description string
	Content     string
	URL         string
	Categories  []string
}

// UnifiedProcessingResult представляет результат объединенной обработки
type UnifiedProcessingResult struct {
	Index      int
	Title      string
	Content    string
	CategoryID int
	Confidence float64
	Error      error
}
