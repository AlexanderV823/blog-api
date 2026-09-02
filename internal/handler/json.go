package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"unicode/utf8"
)

// decodeJSONStrict настраивает строгое и безопасное чтение JSON
func decodeJSONStrict(w http.ResponseWriter, r *http.Request, dst interface{}, maxBytes int64) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return err
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return errors.New("body must contain only a single JSON value")
		}
		return fmt.Errorf("unexpected data after JSON: %w", err)
	}
	return nil
}

// cleanAndValidateString удаляет пробелы и проверяет длину в рунах
func cleanAndValidateString(val string, minRunes, maxRunes int) (string, bool) {
	cleaned := strings.TrimSpace(val)
	runeCount := utf8.RuneCountInString(cleaned)
	if runeCount < minRunes || runeCount > maxRunes {
		return "", false
	}
	return cleaned, true
}

// cleanAndValidateEmail нормализует email и проверяет его стандартными средствами
func cleanAndValidateEmail(email string) (string, bool) {
	cleaned := strings.ToLower(strings.TrimSpace(email))
	addr, err := mail.ParseAddress(cleaned)
	if err != nil {
		return "", false
	}
	if addr.Address == "" || utf8.RuneCountInString(addr.Address) > 100 {
		return "", false
	}
	return addr.Address, true
}

// extractIDFromPath извлекает ID из пути URL
func extractIDFromPath(path, prefix string) string {
	cleaned := strings.TrimPrefix(path, prefix)
	segments := strings.Split(cleaned, "/")
	if len(segments) > 0 {
		return segments[0]
	}
	return ""
}

// ParsePagination извлекает и строго валидирует параметры пагинации.
// Возвращает 400 при неверном формате данных.
func ParsePagination(r *http.Request, defaultLimit, maxLimit int) (int, int, error) {
	query := r.URL.Query()
	limitStr := query.Get("limit")
	offsetStr := query.Get("offset")

	limit := defaultLimit
	var err error
	if limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit <= 0 || limit > maxLimit {
			return 0, 0, fmt.Errorf("invalid limit parameter: must be a positive integer up to %d", maxLimit)
		}
	}

	offset := 0
	if offsetStr != "" {
		offset, err = strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			return 0, 0, errors.New("invalid offset parameter: must be a non-negative integer")
		}
	}

	return limit, offset, nil
}
