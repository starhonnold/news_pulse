#!/usr/bin/env python3
"""
Тестирование FastText классификатора
"""

import sys
import time
from classifier import FastTextNewsClassifier


def test_classifier():
    """Тестирование классификатора на примерах"""
    
    print("=" * 80)
    print("🧪 ТЕСТИРОВАНИЕ FASTTEXT КЛАССИФИКАТОРА")
    print("=" * 80)
    
    # Создание и загрузка классификатора
    print("\n📥 Загрузка классификатора...")
    classifier = FastTextNewsClassifier()
    classifier.load_model()
    
    # Информация о модели
    model_info = classifier.get_model_info()
    print(f"\n📊 Информация о модели:")
    print(f"  Репозиторий: {model_info['repo_id']}")
    print(f"  Файл: {model_info['filename']}")
    print(f"  Загружена: {model_info['is_loaded']}")
    print(f"  FastText категорий: {model_info['categories']}")
    print(f"  Проект категорий: {model_info['target_categories']}")
    
    print("\n" + "=" * 80)
    print("📝 ТЕСТОВЫЕ ПРИМЕРЫ")
    print("=" * 80)
    
    # Тестовые примеры по категориям
    test_cases = [
        {
            'title': 'Футбольный матч Россия - Бразилия завершился со счетом 2:1',
            'description': 'Сборная России одержала победу над командой Бразилии в товарищеском матче',
            'expected_id': 1,
            'expected_name': 'Спорт'
        },
        {
            'title': 'Apple представила новый iPhone 15 Pro с процессором A17',
            'description': 'Компания Apple анонсировала новые смартфоны с улучшенными камерами и производительностью',
            'expected_id': 2,
            'expected_name': 'Технологии'
        },
        {
            'title': 'Ученые открыли новый вид динозавров в Сибири',
            'description': 'Палеонтологи обнаружили останки ранее неизвестного вида динозавров',
            'expected_id': 2,
            'expected_name': 'Технологии'
        },
        {
            'title': 'Президент подписал закон о бюджете на 2024 год',
            'description': 'Федеральный бюджет предусматривает расходы на социальные программы и инфраструктуру',
            'expected_id': 3,
            'expected_name': 'Политика'
        },
        {
            'title': 'На границе произошел вооруженный конфликт',
            'description': 'Столкновения между военными продолжаются уже несколько дней',
            'expected_id': 3,
            'expected_name': 'Политика'
        },
        {
            'title': 'Центробанк повысил ключевую ставку до 16%',
            'description': 'ЦБ РФ принял решение о повышении ключевой ставки для борьбы с инфляцией',
            'expected_id': 4,
            'expected_name': 'Экономика и финансы'
        },
        {
            'title': 'В Москве открылся новый детский сад на 300 мест',
            'description': 'Современное дошкольное учреждение оснащено игровыми площадками и бассейном',
            'expected_id': 5,
            'expected_name': 'Общество'
        },
        {
            'title': 'В Эрмитаже открылась выставка импрессионистов',
            'description': 'Посетители смогут увидеть работы Моне, Ренуара и других художников',
            'expected_id': 5,
            'expected_name': 'Общество'
        },
        {
            'title': 'Врачи назвали лучшие продукты для укрепления иммунитета',
            'description': 'Эксперты рекомендуют включить в рацион цитрусовые, имбирь и мед',
            'expected_id': 5,
            'expected_name': 'Общество'
        },
        {
            'title': 'Топ-10 лучших курортов для отдыха летом',
            'description': 'Эксперты составили рейтинг самых популярных направлений для туристов',
            'expected_id': 5,
            'expected_name': 'Общество'
        },
    ]
    
    # Классификация каждого примера
    correct = 0
    total = len(test_cases)
    
    for i, test_case in enumerate(test_cases, 1):
        text = f"{test_case['title']}. {test_case['description']}"
        result = classifier.classify_text(text)
        
        is_correct = result['category_id'] == test_case['expected_id']
        if is_correct:
            correct += 1
            status = "✅"
        else:
            status = "❌"
        
        print(f"\n{status} Пример {i}:")
        print(f"  Заголовок: {test_case['title'][:60]}...")
        print(f"  Ожидаемая: {test_case['expected_name']} (ID: {test_case['expected_id']})")
        print(f"  FastText: {result['original_category']} (score: {result['original_score']:.4f})")
        print(f"  Результат: {result['category_name']} (ID: {result['category_id']})")
        print(f"  Уверенность: {result['confidence']:.4f}")
    
    # Итоговая статистика
    accuracy = (correct / total) * 100
    print("\n" + "=" * 80)
    print("📊 РЕЗУЛЬТАТЫ ТЕСТИРОВАНИЯ")
    print("=" * 80)
    print(f"Правильных классификаций: {correct}/{total}")
    print(f"Точность: {accuracy:.1f}%")
    print("=" * 80)
    
    # Тест пакетной классификации
    print("\n" + "=" * 80)
    print("📦 ТЕСТ ПАКЕТНОЙ КЛАССИФИКАЦИИ")
    print("=" * 80)
    
    batch_items = [
        {'index': 0, 'title': 'Футбольный матч', 'description': 'Завершился со счетом 2:1'},
        {'index': 1, 'title': 'iPhone 15 Pro', 'description': 'Apple представила новый смартфон'},
        {'index': 2, 'title': 'Центробанк повысил ставку', 'description': 'ЦБ принял решение'},
    ]
    
    batch_results = classifier.classify_batch(batch_items)
    
    for result in batch_results:
        print(f"\nИндекс {result['index']}:")
        print(f"  Категория: {result['category_name']} (ID: {result['category_id']})")
        print(f"  FastText: {result['original_category']} (score: {result['original_score']:.4f})")
    
    # Тест производительности
    print("\n" + "=" * 80)
    print("⚡ ТЕСТ ПРОИЗВОДИТЕЛЬНОСТИ")
    print("=" * 80)
    
    test_text = "Футбольный матч завершился со счетом 3:2 в пользу домашней команды"
    iterations = 1000
    
    print(f"Итераций: {iterations}")
    print("Выполнение...")
    
    start_time = time.time()
    for _ in range(iterations):
        classifier.classify_text(test_text)
    elapsed = time.time() - start_time
    
    avg_time = (elapsed / iterations) * 1000
    throughput = iterations / elapsed
    
    print(f"\nОбщее время: {elapsed:.3f}с")
    print(f"Среднее время: {avg_time:.2f}мс")
    print(f"Пропускная способность: {throughput:.1f} классификаций/сек")
    print("=" * 80)
    
    return classifier, accuracy


if __name__ == "__main__":
    try:
        classifier, accuracy = test_classifier()
        print(f"\n✅ Тестирование завершено успешно! Точность: {accuracy:.1f}%")
        sys.exit(0)
    except Exception as e:
        print(f"\n❌ Ошибка: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)

