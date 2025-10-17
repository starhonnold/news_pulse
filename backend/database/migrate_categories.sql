-- Скрипт миграции категорий новостей
-- Обновляет существующие данные в соответствии с новыми категориями

-- Создаем временную таблицу для маппинга старых категорий на новые
CREATE TEMP TABLE category_mapping (
    old_id INTEGER,
    new_id INTEGER,
    old_name VARCHAR(100),
    new_name VARCHAR(100)
);

-- Заполняем маппинг старых категорий на новые
INSERT INTO category_mapping (old_id, new_id, old_name, new_name) VALUES
-- Старые категории -> Новые категории
(1, 3, 'Политика', 'Политика'),                    -- Политика остается
(2, 4, 'Экономика', 'Экономика и финансы'),        -- Экономика -> Экономика и финансы
(3, 1, 'Спорт', 'Спорт'),                          -- Спорт остается
(4, 2, 'Технологии', 'Технологии'),                -- Технологии остается
(5, 5, 'Культура', 'Общество'),                    -- Культура -> Общество
(6, 5, 'Наука', 'Общество'),                       -- Наука -> Общество
(7, 5, 'Общество', 'Общество'),                    -- Общество остается
(8, 5, 'Происшествия', 'Общество'),                -- Происшествия -> Общество
(9, 5, 'Здоровье', 'Общество'),                    -- Здоровье -> Общество
(10, 5, 'Образование', 'Общество'),                -- Образование -> Общество
(11, 5, 'Международные', 'Общество'),              -- Международные -> Общество
(12, 4, 'Бизнес', 'Экономика и финансы');          -- Бизнес -> Экономика и финансы

-- Обновляем существующие новости с новыми категориями
UPDATE news 
SET category_id = cm.new_id
FROM category_mapping cm
WHERE news.category_id = cm.old_id;

-- Обновляем связи пульсов с категориями
UPDATE pulse_categories 
SET category_id = cm.new_id
FROM category_mapping cm
WHERE pulse_categories.category_id = cm.old_id;

-- Удаляем старые категории (кроме тех, которые остались)
DELETE FROM categories 
WHERE id NOT IN (1, 2, 3, 4, 5);

-- Обновляем ID категорий в таблице categories
-- Сначала создаем новые записи с правильными ID
INSERT INTO categories (id, name, slug, color, icon, description, is_active, created_at, updated_at)
SELECT 
    cm.new_id,
    cm.new_name,
    CASE cm.new_id
        WHEN 1 THEN 'sport'
        WHEN 2 THEN 'tech'
        WHEN 3 THEN 'politics'
        WHEN 4 THEN 'economy'
        WHEN 5 THEN 'society'
    END as slug,
    CASE cm.new_id
        WHEN 1 THEN 'blue-6'
        WHEN 2 THEN 'purple-6'
        WHEN 3 THEN 'red-6'
        WHEN 4 THEN 'green-6'
        WHEN 5 THEN 'teal-6'
    END as color,
    CASE cm.new_id
        WHEN 1 THEN 'sports_soccer'
        WHEN 2 THEN 'computer'
        WHEN 3 THEN 'gavel'
        WHEN 4 THEN 'trending_up'
        WHEN 5 THEN 'people'
    END as icon,
    CASE cm.new_id
        WHEN 1 THEN 'Спортивные новости и события'
        WHEN 2 THEN 'Технологические новости и инновации'
        WHEN 3 THEN 'Политические новости и события'
        WHEN 4 THEN 'Экономические новости и финансы'
        WHEN 5 THEN 'Общественные события и социальные вопросы'
    END as description,
    true as is_active,
    CURRENT_TIMESTAMP as created_at,
    CURRENT_TIMESTAMP as updated_at
FROM category_mapping cm
WHERE cm.new_id IN (1, 2, 3, 4, 5);

-- Обновляем внешние ключи в таблице news
-- Сначала удаляем внешние ключи
ALTER TABLE news DROP CONSTRAINT IF EXISTS news_category_id_fkey;

-- Обновляем ID категорий в таблице news
UPDATE news 
SET category_id = cm.new_id
FROM category_mapping cm
WHERE news.category_id = cm.old_id;

-- Восстанавливаем внешний ключ
ALTER TABLE news 
ADD CONSTRAINT news_category_id_fkey 
FOREIGN KEY (category_id) REFERENCES categories(id);

-- Обновляем внешние ключи в таблице pulse_categories
-- Сначала удаляем внешние ключи
ALTER TABLE pulse_categories DROP CONSTRAINT IF EXISTS pulse_categories_category_id_fkey;

-- Обновляем ID категорий в таблице pulse_categories
UPDATE pulse_categories 
SET category_id = cm.new_id
FROM category_mapping cm
WHERE pulse_categories.category_id = cm.old_id;

-- Восстанавливаем внешний ключ
ALTER TABLE pulse_categories 
ADD CONSTRAINT pulse_categories_category_id_fkey 
FOREIGN KEY (category_id) REFERENCES categories(id);

-- Обновляем последовательность для categories
SELECT setval('categories_id_seq', 5, true);

-- Выводим статистику миграции
SELECT 
    'Миграция завершена' as status,
    COUNT(*) as total_news_updated
FROM news 
WHERE category_id IS NOT NULL;

-- Показываем распределение новостей по новым категориям
SELECT 
    c.name as category_name,
    COUNT(n.id) as news_count
FROM categories c
LEFT JOIN news n ON c.id = n.category_id
GROUP BY c.id, c.name
ORDER BY c.id;
