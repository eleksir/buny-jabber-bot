package jabber

import (
	"regexp"
	"strings"
)

// Переменные для простенького нормализатора текста.
var (
	// Знаки препинания раз.
	pMarks = []string{".", ",", "!", "@", "#", "$", "%", "^", "&", "*", "(", ")", "{", "}", "<", ">", "[", "]", "\\"}

	// Знаки препинания два-с.
	pMarks2 = []string{"-", "_", "+", "=", ":", ";", "'", "`", "~", "\""}

	// Символы новой строки.
	newLines = []string{"\n", "\r", "\n\r", "\r\n"}
)

// Нормализует текстовый буфер. Убирает начальные и конечные пробельные символы, заменяет сносы строк пробелами, удаляет
// знаки препинания, схлопывает повторяющиеся пробелы в один символ.
func nString(buf string) string {
	// Убираем пробелы вначале текстового буфера и в конце.
	buf = strings.Trim(buf, "\n\r\t ")

	// Убираем знаки препинания.
	for _, pMark := range pMarks {
		buf = strings.ReplaceAll(buf, pMark, "")
	}

	for _, pMark := range pMarks2 {
		buf = strings.ReplaceAll(buf, pMark, "")
	}

	// Замещаем сносы строки пробелами.
	for _, newline := range newLines {
		buf = strings.ReplaceAll(buf, newline, " ")
	}

	// Схлопываем повторяющиеся пробелы.
	buf = regexp.MustCompile(`\s+`).ReplaceAllString(buf, " ")

	return buf
}

// Обёртка для nString, возвращает нормализованную строку в нижнем регистре.
func nStringLower(buf string) string { //nolint:unused
	return strings.ToLower(nString(buf))
}

// Обёртка для nString, возвращает нормализованную строку в верхнем регистре.
func nStringUpper(buf string) string {
	return strings.ToUpper(nString(buf))
}

/* vim: set ft=go noet ai ts=4 sw=4 sts=4: */
