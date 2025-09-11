-- PostgreSQL schema for News Pulse application
-- Схема базы данных для приложения "Пульс Новостей"

-- Создание базы данных (выполнить отдельно)
-- CREATE DATABASE news_pulse;
-- \c news_pulse;

-- Включение расширений
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Таблица стран
CREATE TABLE countries (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    code VARCHAR(2) NOT NULL UNIQUE, -- ISO код страны (RU, BY, KZ и т.д.)
    flag_emoji VARCHAR(10) NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Таблица категорий новостей
CREATE TABLE categories (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    slug VARCHAR(100) NOT NULL UNIQUE,
    color VARCHAR(20) DEFAULT 'blue-6', -- Цвет для UI
    icon VARCHAR(50) DEFAULT 'article', -- Material Design иконка
    description TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Таблица источников новостей
CREATE TABLE news_sources (
    id SERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    domain VARCHAR(200) NOT NULL,
    rss_url VARCHAR(500) NOT NULL,
    website_url VARCHAR(500),
    country_id INTEGER NOT NULL REFERENCES countries(id),
    language VARCHAR(10) DEFAULT 'ru',
    description TEXT,
    logo_url VARCHAR(500),
    is_active BOOLEAN DEFAULT TRUE,
    last_parsed_at TIMESTAMP,
    parse_interval_minutes INTEGER DEFAULT 10, -- Интервал парсинга в минутах
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(rss_url)
);

-- Таблица новостей
CREATE TABLE news (
    id SERIAL PRIMARY KEY,
    title VARCHAR(1000) NOT NULL,
    description TEXT,
    content TEXT,
    url VARCHAR(1000) NOT NULL,
    image_url VARCHAR(1000),
    author VARCHAR(200),
    source_id INTEGER NOT NULL REFERENCES news_sources(id),
    category_id INTEGER REFERENCES categories(id),
    published_at TIMESTAMP NOT NULL,
    parsed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    relevance_score DECIMAL(3,2) DEFAULT 0.5, -- Оценка релевантности 0.0-1.0
    view_count INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(url, source_id) -- Предотвращаем дублирование по URL и источнику
);

-- Таблица тегов
CREATE TABLE tags (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    slug VARCHAR(100) NOT NULL UNIQUE,
    usage_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Связь новостей с тегами (многие ко многим)
CREATE TABLE news_tags (
    news_id INTEGER NOT NULL REFERENCES news(id) ON DELETE CASCADE,
    tag_id INTEGER NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (news_id, tag_id)
);

-- Таблица пользователей (для будущей SMS авторизации)
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    phone_number VARCHAR(20) UNIQUE, -- Для SMS авторизации
    phone_verified BOOLEAN DEFAULT FALSE,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    email VARCHAR(255),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Таблица SMS кодов (для будущей SMS авторизации)
CREATE TABLE sms_codes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    phone_number VARCHAR(20) NOT NULL,
    code VARCHAR(6) NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    attempts INTEGER DEFAULT 0,
    is_used BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Таблица пульсов пользователей
CREATE TABLE user_pulses (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(200) NOT NULL,
    description TEXT,
    keywords TEXT, -- Ключевые слова через запятую
    is_active BOOLEAN DEFAULT TRUE,
    is_default BOOLEAN DEFAULT FALSE, -- Является ли пульс дефолтным для пользователя
    news_count INTEGER DEFAULT 0, -- Кеш количества новостей
    refresh_interval_min INTEGER DEFAULT 10, -- Интервал обновления в минутах
    last_refreshed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, -- Время последнего обновления
    last_updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Связь пульсов с странами (многие ко многим)
CREATE TABLE pulse_countries (
    pulse_id UUID NOT NULL REFERENCES user_pulses(id) ON DELETE CASCADE,
    country_id INTEGER NOT NULL REFERENCES countries(id) ON DELETE CASCADE,
    PRIMARY KEY (pulse_id, country_id)
);

-- Связь пульсов с источниками (многие ко многим)
CREATE TABLE pulse_sources (
    pulse_id UUID NOT NULL REFERENCES user_pulses(id) ON DELETE CASCADE,
    source_id INTEGER NOT NULL REFERENCES news_sources(id) ON DELETE CASCADE,
    PRIMARY KEY (pulse_id, source_id)
);

-- Связь пульсов с категориями (многие ко многим)
CREATE TABLE pulse_categories (
    pulse_id UUID NOT NULL REFERENCES user_pulses(id) ON DELETE CASCADE,
    category_id INTEGER NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    PRIMARY KEY (pulse_id, category_id)
);

-- Таблица связи пульсов с новостями (для персонализированных лент)
CREATE TABLE pulse_news (
    pulse_id UUID NOT NULL REFERENCES user_pulses(id) ON DELETE CASCADE,
    news_id INTEGER NOT NULL REFERENCES news(id) ON DELETE CASCADE,
    relevance_score DECIMAL(3,2) DEFAULT 0.5, -- Релевантность новости для пульса
    match_reason TEXT, -- Причина попадания в ленту (категория, ключевые слова и т.д.)
    added_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (pulse_id, news_id)
);

-- Таблица для логирования парсинга RSS
CREATE TABLE parsing_logs (
    id SERIAL PRIMARY KEY,
    source_id INTEGER NOT NULL REFERENCES news_sources(id),
    status VARCHAR(20) NOT NULL, -- success, error, timeout
    news_count INTEGER DEFAULT 0, -- Количество спарсенных новостей
    error_message TEXT,
    execution_time_ms INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Индексы для оптимизации производительности
CREATE INDEX idx_news_published_at ON news(published_at DESC);
CREATE INDEX idx_news_source_id ON news(source_id);
CREATE INDEX idx_news_category_id ON news(category_id);
CREATE INDEX idx_news_relevance_score ON news(relevance_score DESC);
CREATE INDEX idx_news_title_gin ON news USING gin(to_tsvector('russian', title));
CREATE INDEX idx_news_description_gin ON news USING gin(to_tsvector('russian', description));
CREATE INDEX idx_news_content_gin ON news USING gin(to_tsvector('russian', content));
CREATE INDEX idx_news_url ON news(url);

CREATE INDEX idx_news_sources_country_id ON news_sources(country_id);
CREATE INDEX idx_news_sources_active ON news_sources(is_active);
CREATE INDEX idx_news_sources_last_parsed ON news_sources(last_parsed_at);

CREATE INDEX idx_user_pulses_user_id ON user_pulses(user_id);
CREATE INDEX idx_user_pulses_active ON user_pulses(is_active);

CREATE INDEX idx_pulse_news_pulse_id ON pulse_news(pulse_id);
CREATE INDEX idx_pulse_news_news_id ON pulse_news(news_id);
CREATE INDEX idx_pulse_news_relevance_score ON pulse_news(relevance_score DESC);
CREATE INDEX idx_pulse_news_added_at ON pulse_news(added_at DESC);

CREATE INDEX idx_sms_codes_phone ON sms_codes(phone_number);
CREATE INDEX idx_sms_codes_expires_at ON sms_codes(expires_at);

-- Триггеры для автоматического обновления timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Применяем триггеры к таблицам
CREATE TRIGGER update_countries_updated_at BEFORE UPDATE ON countries FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_categories_updated_at BEFORE UPDATE ON categories FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_news_sources_updated_at BEFORE UPDATE ON news_sources FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_news_updated_at BEFORE UPDATE ON news FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_user_pulses_updated_at BEFORE UPDATE ON user_pulses FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Функция для очистки старых SMS кодов
CREATE OR REPLACE FUNCTION cleanup_expired_sms_codes()
RETURNS void AS $$
BEGIN
    DELETE FROM sms_codes 
    WHERE expires_at < CURRENT_TIMESTAMP - INTERVAL '1 day';
END;
$$ LANGUAGE plpgsql;

-- Функция для подсчета новостей в пульсе
CREATE OR REPLACE FUNCTION count_pulse_news(pulse_uuid UUID)
RETURNS INTEGER AS $$
DECLARE
    news_count INTEGER;
BEGIN
    WITH pulse_filters AS (
        SELECT 
            p.keywords,
            ARRAY_AGG(DISTINCT pc.country_id) as country_ids,
            ARRAY_AGG(DISTINCT ps.source_id) as source_ids,
            ARRAY_AGG(DISTINCT pcat.category_id) as category_ids
        FROM user_pulses p
        LEFT JOIN pulse_countries pc ON p.id = pc.pulse_id
        LEFT JOIN pulse_sources ps ON p.id = ps.pulse_id
        LEFT JOIN pulse_categories pcat ON p.id = pcat.pulse_id
        WHERE p.id = pulse_uuid
        GROUP BY p.id, p.keywords
    )
    SELECT COUNT(*)
    INTO news_count
    FROM news n
    JOIN news_sources ns ON n.source_id = ns.id
    JOIN pulse_filters pf ON (
        (pf.country_ids IS NULL OR ns.country_id = ANY(pf.country_ids))
        AND (pf.source_ids IS NULL OR n.source_id = ANY(pf.source_ids))
        AND (pf.category_ids IS NULL OR n.category_id = ANY(pf.category_ids))
        AND (
            pf.keywords IS NULL 
            OR pf.keywords = '' 
            OR to_tsvector('russian', n.title || ' ' || COALESCE(n.description, '')) @@ plainto_tsquery('russian', pf.keywords)
        )
    )
    WHERE n.is_active = true;
    
    RETURN COALESCE(news_count, 0);
END;
$$ LANGUAGE plpgsql;
