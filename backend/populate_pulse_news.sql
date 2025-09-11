-- Функция для заполнения pulse_news на основе существующих пульсов
CREATE OR REPLACE FUNCTION populate_pulse_news() RETURNS void AS $$
DECLARE
    pulse_record RECORD;
    news_record RECORD;
    match_reason VARCHAR(50);
    relevance_score NUMERIC(3,2);
BEGIN
    -- Очищаем существующие данные
    DELETE FROM pulse_news;
    
    -- Проходим по всем пульсам
    FOR pulse_record IN 
        SELECT up.id, up.keywords, up.refresh_interval_min
        FROM user_pulses up
        WHERE up.is_active = true
    LOOP
        -- Проходим по всем новостям
        FOR news_record IN 
            SELECT n.id, n.title, n.description, n.source_id, n.category_id, n.published_at, n.relevance_score
            FROM news n
            WHERE n.is_active = true
            AND n.published_at >= NOW() - INTERVAL '7 days' -- Только последние 7 дней
        LOOP
            match_reason := NULL;
            relevance_score := news_record.relevance_score;
            
            -- Проверяем совпадение по категории (только если есть категории в пульсе)
            IF EXISTS (
                SELECT 1 FROM pulse_categories pc 
                WHERE pc.pulse_id = pulse_record.id
            ) THEN
                IF EXISTS (
                    SELECT 1 FROM pulse_categories pc 
                    WHERE pc.pulse_id = pulse_record.id 
                    AND pc.category_id = news_record.category_id
                ) THEN
                    match_reason := 'category';
                    relevance_score := relevance_score + 0.2;
                END IF;
            ELSE
                -- Если нет категорий в пульсе, берем все новости
                match_reason := 'no_categories';
                relevance_score := relevance_score + 0.1;
            END IF;
            
            -- Проверяем совпадение по источнику
            IF EXISTS (
                SELECT 1 FROM pulse_sources ps 
                WHERE ps.pulse_id = pulse_record.id 
                AND ps.source_id = news_record.source_id
            ) THEN
                match_reason := COALESCE(match_reason, 'source');
                relevance_score := relevance_score + 0.1;
            END IF;
            
            -- Проверяем совпадение по ключевым словам
            IF pulse_record.keywords IS NOT NULL AND pulse_record.keywords != '' THEN
                IF news_record.title ILIKE '%' || pulse_record.keywords || '%' 
                   OR news_record.description ILIKE '%' || pulse_record.keywords || '%' THEN
                    match_reason := COALESCE(match_reason, 'keyword');
                    relevance_score := relevance_score + 0.3;
                END IF;
            END IF;
            
            -- Если есть хотя бы одно совпадение, добавляем в pulse_news
            IF match_reason IS NOT NULL THEN
                INSERT INTO pulse_news (pulse_id, news_id, match_reason, relevance_score)
                VALUES (pulse_record.id, news_record.id, match_reason, LEAST(relevance_score, 1.0))
                ON CONFLICT (pulse_id, news_id) DO NOTHING;
            END IF;
        END LOOP;
    END LOOP;
END;
$$ LANGUAGE plpgsql;
