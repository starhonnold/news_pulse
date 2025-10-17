-- Seed data for News Pulse application
-- Начальные данные для приложения "Пульс Новостей"

-- Вставка стран
INSERT INTO countries (name, code, flag_emoji) VALUES
('Россия', 'RU', '🇷🇺'),
('Беларусь', 'BY', '🇧🇾'),
('Казахстан', 'KZ', '🇰🇿'),
('Украина', 'UA', '🇺🇦'),
('Узбекистан', 'UZ', '🇺🇿'),
('Кыргызстан', 'KG', '🇰🇬'),
('Армения', 'AM', '🇦🇲'),
('Азербайджан', 'AZ', '🇦🇿'),
('Таджикистан', 'TJ', '🇹🇯'),
('Туркменистан', 'TM', '🇹🇲'),
('Молдова', 'MD', '🇲🇩'),
('Грузия', 'GE', '🇬🇪');

-- Вставка категорий новостей
INSERT INTO categories (name, slug, color, icon, description) VALUES
('Спорт', 'sport', 'blue-6', 'sports_soccer', 'Спортивные новости и события'),
('Технологии', 'tech', 'purple-6', 'computer', 'Технологические новости и инновации'),
('Политика', 'politics', 'red-6', 'gavel', 'Политические новости и события'),
('Экономика и финансы', 'economy', 'green-6', 'trending_up', 'Экономические новости и финансы'),
('Общество', 'society', 'teal-6', 'people', 'Общественные события и социальные вопросы');

-- Вставка источников новостей

-- Российские источники
INSERT INTO news_sources (name, domain, rss_url, website_url, country_id, description, logo_url) VALUES
-- Основные федеральные агентства
('РИА Новости', 'ria.ru', 'https://ria.ru/export/rss2/archive/index.xml', 'https://ria.ru', 1, 'Российское информационное агентство', 'https://ria.ru/favicon.ico'),
('ТАСС', 'tass.ru', 'https://tass.ru/rss/v2.xml', 'https://tass.ru', 1, 'Информационное агентство России ТАСС', 'https://tass.ru/favicon.ico'),
('Интерфакс', 'interfax.ru', 'https://www.interfax.ru/rss.asp', 'https://www.interfax.ru', 1, 'Российское информационное агентство', 'https://www.interfax.ru/favicon.ico'),

-- Интернет-издания
('Lenta.ru', 'lenta.ru', 'https://lenta.ru/rss', 'https://lenta.ru', 1, 'Интернет-издание', 'https://lenta.ru/favicon.ico'),
('Газета.Ru', 'gazeta.ru', 'https://www.gazeta.ru/export/rss/lenta.xml', 'https://www.gazeta.ru', 1, 'Ежедневное интернет-издание', 'https://www.gazeta.ru/favicon.ico'),
('РБК', 'rbc.ru', 'https://rssexport.rbc.ru/rbcnews/news/30/full.rss', 'https://www.rbc.ru', 1, 'РосБизнесКонсалтинг', 'https://www.rbc.ru/favicon.ico'),

-- Деловые издания
('Коммерсант', 'kommersant.ru', 'https://www.kommersant.ru/RSS/news.xml', 'https://www.kommersant.ru', 1, 'Ежедневная общенациональная деловая газета', 'https://www.kommersant.ru/favicon.ico'),
('Ведомости', 'vedomosti.ru', 'https://www.vedomosti.ru/rss/news', 'https://www.vedomosti.ru', 1, 'Ежедневная деловая газета', 'https://www.vedomosti.ru/favicon.ico'),

-- Международные русскоязычные
('RT на русском', 'russian.rt.com', 'https://russian.rt.com/rss', 'https://russian.rt.com', 1, 'RT - международный информационный канал', 'https://russian.rt.com/favicon.ico'),
('Sputnik', 'sputniknews.ru', 'https://sputniknews.ru/export/rss2/archive/index.xml', 'https://sputniknews.ru', 1, 'Международное информационное агентство', 'https://sputniknews.ru/favicon.ico');

-- Белорусские источники
INSERT INTO news_sources (name, domain, rss_url, website_url, country_id, description, logo_url) VALUES
('БЕЛТА', 'belta.by', 'https://www.belta.by/rss/', 'https://www.belta.by', 2, 'Белорусское телеграфное агентство', 'https://www.belta.by/favicon.ico'),
('Советская Беларусь', 'sb.by', 'https://www.sb.by/rss/', 'https://www.sb.by', 2, 'Республиканская ежедневная газета', 'https://www.sb.by/favicon.ico'),
('Белорусские новости', 'belarusnews.by', 'https://belarusnews.by/rss.xml', 'https://belarusnews.by', 2, 'Информационный портал Беларуси', 'https://belarusnews.by/favicon.ico');

-- Казахстанские источники
INSERT INTO news_sources (name, domain, rss_url, website_url, country_id, description, logo_url) VALUES
('Казинформ', 'inform.kz', 'https://www.inform.kz/ru/rss', 'https://www.inform.kz', 3, 'Казахское информационное агентство', 'https://www.inform.kz/favicon.ico'),
('Tengrinews', 'tengrinews.kz', 'https://tengrinews.kz/rss/', 'https://tengrinews.kz', 3, 'Информационный портал Казахстана', 'https://tengrinews.kz/favicon.ico'),
('Nur.kz', 'nur.kz', 'https://www.nur.kz/rss/', 'https://www.nur.kz', 3, 'Казахстанский информационный портал', 'https://www.nur.kz/favicon.ico');

-- Украинские источники (русскоязычные)
INSERT INTO news_sources (name, domain, rss_url, website_url, country_id, description, logo_url) VALUES
('УНИАН', 'unian.net', 'https://rss.unian.net/site/news_rus.rss', 'https://www.unian.net', 4, 'Украинское независимое информационное агентство', 'https://www.unian.net/favicon.ico'),
('Корреспондент.net', 'korrespondent.net', 'https://korrespondent.net/rss/', 'https://korrespondent.net', 4, 'Украинское интернет-издание', 'https://korrespondent.net/favicon.ico');

-- Узбекистанские источники
INSERT INTO news_sources (name, domain, rss_url, website_url, country_id, description, logo_url) VALUES
('УзА', 'uza.uz', 'https://www.uza.uz/ru/rss/all', 'https://www.uza.uz', 5, 'Национальное информационное агентство Узбекистана', 'https://www.uza.uz/favicon.ico'),
('Газета.uz', 'gazeta.uz', 'https://www.gazeta.uz/ru/rss/', 'https://www.gazeta.uz', 5, 'Узбекское интернет-издание', 'https://www.gazeta.uz/favicon.ico');

-- Кыргызстанские источники
INSERT INTO news_sources (name, domain, rss_url, website_url, country_id, description, logo_url) VALUES
('Kabar', 'kabar.kg', 'https://kabar.kg/rss/all_news.xml', 'https://kabar.kg', 6, 'Государственное информационное агентство КР', 'https://kabar.kg/favicon.ico'),
('24.kg', '24.kg', 'https://24.kg/rss/', 'https://24.kg', 6, 'Информационное агентство Кыргызстана', 'https://24.kg/favicon.ico');

-- Армянские источники
INSERT INTO news_sources (name, domain, rss_url, website_url, country_id, description, logo_url) VALUES
('Armenpress', 'armenpress.am', 'https://armenpress.am/rus/rss/', 'https://armenpress.am', 7, 'Национальное информационное агентство Армении', 'https://armenpress.am/favicon.ico'),
('NEWS.am', 'news.am', 'https://news.am/rus/rss/', 'https://news.am', 7, 'Информационное агентство Армении', 'https://news.am/favicon.ico');

-- Азербайджанские источники
INSERT INTO news_sources (name, domain, rss_url, website_url, country_id, description, logo_url) VALUES
('АЗЕРТАДЖ', 'azertag.az', 'https://azertag.az/ru/rss', 'https://azertag.az', 8, 'Государственное информационное агентство Азербайджана', 'https://azertag.az/favicon.ico'),
('Trend', 'trend.az', 'https://www.trend.az/rss/news_ru.xml', 'https://www.trend.az', 8, 'Информационное агентство Азербайджана', 'https://www.trend.az/favicon.ico');

-- Таджикистанские источники
INSERT INTO news_sources (name, domain, rss_url, website_url, country_id, description, logo_url) VALUES
('Ховар', 'khovar.tj', 'https://khovar.tj/rus/rss/', 'https://khovar.tj', 9, 'Национальное информационное агентство Таджикистана', 'https://khovar.tj/favicon.ico'),
('Asia-Plus', 'news.tj', 'https://news.tj/ru/rss', 'https://news.tj', 9, 'Информационное агентство Таджикистана', 'https://news.tj/favicon.ico');

-- Молдавские источники
INSERT INTO news_sources (name, domain, rss_url, website_url, country_id, description, logo_url) VALUES
('Moldpres', 'moldpres.md', 'https://www.moldpres.md/ru/rss', 'https://www.moldpres.md', 11, 'Национальное информационное агентство Молдовы', 'https://www.moldpres.md/favicon.ico'),
('NewsMaker', 'newsmaker.md', 'https://newsmaker.md/rss/', 'https://newsmaker.md', 11, 'Информационное агентство Молдовы', 'https://newsmaker.md/favicon.ico');

-- Грузинские источники (русскоязычные)
INSERT INTO news_sources (name, domain, rss_url, website_url, country_id, description, logo_url) VALUES
('Sputnik Грузия', 'sputnik-georgia.ru', 'https://sputnik-georgia.ru/export/rss2/archive/index.xml', 'https://sputnik-georgia.ru', 12, 'Информационное агентство Sputnik в Грузии', 'https://sputnik-georgia.ru/favicon.ico'),
('1TV.ge', '1tv.ge', 'https://1tv.ge/ru/rss/', 'https://1tv.ge', 12, 'Первый канал Грузии', 'https://1tv.ge/favicon.ico');

-- Вставка некоторых базовых тегов
INSERT INTO tags (name, slug) VALUES
('спорт', 'sport'),
('технологии', 'technology'),
('политика', 'politics'),
('экономика', 'economy'),
('финансы', 'finance'),
('общество', 'society'),
('футбол', 'football'),
('хоккей', 'hockey'),
('баскетбол', 'basketball'),
('теннис', 'tennis'),
('олимпиада', 'olympics'),
('чемпионат', 'championship'),
('искусственный интеллект', 'ai'),
('блокчейн', 'blockchain'),
('криптовалюты', 'cryptocurrency'),
('стартапы', 'startups'),
('инвестиции', 'investments'),
('недвижимость', 'real-estate'),
('инновации', 'innovations'),
('программирование', 'programming'),
('гаджеты', 'gadgets'),
('смартфоны', 'smartphones'),
('электромобили', 'electric-cars'),
('космос', 'space'),
('выборы', 'elections'),
('правительство', 'government'),
('парламент', 'parliament'),
('президент', 'president'),
('министр', 'minister'),
('закон', 'law'),
('реформа', 'reform'),
('бюджет', 'budget'),
('инфляция', 'inflation'),
('валюты', 'currencies'),
('банки', 'banks'),
('акции', 'stocks'),
('кризис', 'crisis'),
('безработица', 'unemployment'),
('пенсии', 'pensions'),
('социальные сети', 'social-networks'),
('образование', 'education'),
('здравоохранение', 'healthcare'),
('экология', 'ecology'),
('транспорт', 'transport'),
('культура', 'culture'),
('искусство', 'art'),
('музыка', 'music'),
('кино', 'cinema'),
('театр', 'theater'),
('литература', 'literature');

-- DEPRECATED: Тестовые данные удалены
-- НЕ ИСПОЛЬЗОВАТЬ В PRODUCTION!
-- Тестовые пользователи и пульсы должны создаваться через API

-- Обновляем счетчики использования тегов
UPDATE tags SET usage_count = 1 WHERE slug IN ('sport', 'technology', 'politics', 'economy', 'finance', 'society');

-- Комментарии для разработчиков
/*
Примечания по использованию:

1. RSS URL могут изменяться, поэтому их нужно периодически проверять
2. Некоторые сайты могут блокировать частые запросы - используйте разумные интервалы
3. Для production окружения рекомендуется настроить User-Agent и другие заголовки
4. Проверяйте кодировку RSS лент - большинство используют UTF-8
5. Некоторые RSS ленты могут требовать аутентификации или иметь ограничения по IP

Полезные команды для проверки RSS:
- curl -I "URL" - проверить заголовки
- curl "URL" | head -50 - посмотреть начало RSS ленты
- xmllint --format "URL" - отформатировать XML для чтения

Для мониторинга:
- Регулярно проверяйте parsing_logs на ошибки
- Мониторьте last_parsed_at в news_sources
- Следите за размером таблицы news и настройте архивацию старых записей
*/
