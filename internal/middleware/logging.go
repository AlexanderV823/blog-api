package middleware

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"time"
)

type LoggingMiddleware struct {
	logger *log.Logger
}

func NewLoggingMiddleware(logger *log.Logger) *LoggingMiddleware {
	return &LoggingMiddleware{
		logger: logger,
	}
}

// Recovery перехватывает паники и возвращает 500
func (m *LoggingMiddleware) Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				m.logger.Printf("[PANIC] %v\n%s", err, debug.Stack())
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":"Internal server error occurred"}`))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// RequestID добавляет уникальный ID для трассировки
func (m *LoggingMiddleware) RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Генерация простейшего RequestID на базе текущего времени (для демонстрации)
		reqID := fmt.Sprintf("%d", time.Now().UnixNano())
		w.Header().Set("X-Request-ID", reqID)
		next.ServeHTTP(w, r)
	})
}

// CORS настраивает заголовки для работы с фронтендом
func (m *LoggingMiddleware) CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Разрешаем запросы с любых доменов (для продакшена лучше указать конкретные домены)
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// responseWriter — обертка для перехвата статус-кода и размера ответа
type responseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.status == 0 {
		rw.status = http.StatusOK
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.size += n
	return n, err
}

// Logger записывает детальную информацию о каждом HTTP-запросе с учетом IP-адреса клиента
func (m *LoggingMiddleware) Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w}

		next.ServeHTTP(rw, r)

		// 1. Извлекаем реальный IP-адрес клиента
		// Сначала проверяем заголовок X-Forwarded-For (на случай использования Nginx/Cloudflare)
		ip := r.Header.Get("X-Forwarded-For")
		if ip == "" {
			// Если заголовка нет, берем прямой сетевой адрес из запроса
			ip = r.RemoteAddr
		}

		// 2. Выводим лог, добавляя значение IP в начало записи
		m.logger.Printf(
			"[INFO] IP: %s | %s %s | Status: %d | Size: %d bytes | Duration: %s",
			ip, r.Method, r.URL.Path, rw.status, rw.size, time.Since(start),
		)
	})
}
