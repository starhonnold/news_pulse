package utils

import "strings"

// CleanText очищает текст от непечатаемых символов
func CleanText(text string) string {
	if text == "" {
		return text
	}

	// Удаляем непечатаемые символы и управляющие символы
	var result strings.Builder
	for _, r := range text {
		// Проверяем, является ли символ печатаемым
		if IsPrintableRune(r) {
			result.WriteRune(r)
		} else if r == '\n' || r == '\r' || r == '\t' {
			// Сохраняем основные пробельные символы
			result.WriteRune(r)
		} else {
			// Заменяем непечатаемые символы на пробел
			result.WriteRune(' ')
		}
	}

	// Очищаем множественные пробелы
	cleaned := strings.TrimSpace(result.String())
	cleaned = strings.ReplaceAll(cleaned, "  ", " ")

	return cleaned
}

// IsPrintableRune проверяет, является ли руна печатаемой
func IsPrintableRune(r rune) bool {
	// Проверяем диапазоны печатаемых символов
	if r >= 32 && r <= 126 {
		return true // ASCII печатаемые символы
	}
	if r >= 160 && r <= 1114111 {
		return true // Unicode печатаемые символы (включая кириллицу)
	}
	return false
}
