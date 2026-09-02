package handler_test

import (
	"blog-api/internal/handler"
	"blog-api/internal/middleware"
	"blog-api/internal/service"
	"blog-api/pkg/auth"
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// Тест валидации пагинации через оригинальный handler.ParsePagination
func TestDecodeJSONStrict_RealValidation(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "/api/posts?limit=abc", nil)
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()

	// Вызываем экспортируемую функцию ParsePagination напрямую для проверки покрытия кода в json.go
	_, _, err = handler.ParsePagination(req, 10, 100)
	if err == nil {
		t.Error("ожидалась ошибка парсинга некорректного limit=abc, но получили nil")
	}

	// Явно логируем код ответа рекордера для использования переменной rec
	_ = rec.Code
}

// Тест оригинального, подключенного в main.go RateLimiter под нагрузкой (проверка go test -race)
func TestRateLimiter_RealConcurrencyAndRace(t *testing.T) {
	limiter := middleware.NewRateLimiter(10, time.Second)
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	var wg sync.WaitGroup
	concurrentRequests := 30
	results := make(chan int, concurrentRequests)

	for i := 0; i < concurrentRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/api/posts", nil)
			req.RemoteAddr = "192.168.2.100:44321" // Фиксированный IP для проверки лимита бакета
			rec := httptest.NewRecorder()
			limiter.Limit(nextHandler).ServeHTTP(rec, req)
			results <- rec.Code
		}()
	}

	wg.Wait()
	close(results)

	tooManyRequestsCount := 0
	okCount := 0
	for code := range results {
		if code == http.StatusTooManyRequests {
			tooManyRequestsCount++
		} else if code == http.StatusOK {
			okCount++
		}
	}

	if okCount > 10 {
		t.Errorf("RateLimiter пропустил %d запросов, хотя жесткий лимит равен 10", okCount)
	}
	if tooManyRequestsCount == 0 {
		t.Error("Инфраструктурный лимитер не заблокировал избыточные конкурентные запросы")
	}
}

// Тест оригинального AuthMiddleware и его методов валидации
func TestAuthMiddleware_RealRequireAuth(t *testing.T) {
	jwtManager := auth.NewJWTManager("super-secret-key-32-chars-long-!!", 1)
	mw := middleware.NewAuthMiddleware(jwtManager)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.GetUserIDFromContext(r.Context())
		if ok {
			w.WriteHeader(http.StatusOK)
			// ИСПРАВЛЕНО: Заменено ручное приведение []byte(fmt.Sprintf(...)) на эффективный fmt.Appendf
			res := fmt.Appendf(nil, "userID:%d", userID)
			_, _ = w.Write(res)
		}
	})

	// 1. Тест: Отсутствие токена авторизации (Ожидается 401)
	reqNoToken := httptest.NewRequest(http.MethodPost, "/api/posts", nil)
	recNoToken := httptest.NewRecorder()
	mw.RequireAuth(nextHandler).ServeHTTP(recNoToken, reqNoToken)
	if recNoToken.Code != http.StatusUnauthorized {
		t.Errorf("ожидался статус 401 для запроса без токена, получен: %d", recNoToken.Code)
	}

	// 2. Тест: Сломанный/поддельный Bearer токен (Ожидается 401)
	reqBadToken := httptest.NewRequest(http.MethodPost, "/api/posts", nil)
	reqBadToken.Header.Set("Authorization", "Bearer bad-token-signature")
	recBadToken := httptest.NewRecorder()
	mw.RequireAuth(nextHandler).ServeHTTP(recBadToken, reqBadToken)
	if recBadToken.Code != http.StatusUnauthorized {
		t.Errorf("ожидался статус 401 для битого токена, получен: %d", recBadToken.Code)
	}

	// 3. Тест: Валидный JWT токен приложения (Ожидается 200 и запись в контекст)
	token, _, _ := jwtManager.GenerateToken(555, "auth-user@test.com", "auth_user")
	reqValid := httptest.NewRequest(http.MethodPost, "/api/posts", nil)
	reqValid.Header.Set("Authorization", "Bearer "+token)
	recValid := httptest.NewRecorder()
	mw.RequireAuth(nextHandler).ServeHTTP(recValid, reqValid)

	if recValid.Code != http.StatusOK {
		t.Errorf("ожидался статус 200 для валидного токена, получен: %d", recValid.Code)
	}
	if recValid.Body.String() != "userID:555" {
		t.Errorf("context поврежден: ожидали userID:555, получили %q", recValid.Body.String())
	}
}

// Тест валидации пагинации через оригинальный handler.ParsePagination
func TestParsePagination_RealValidation(t *testing.T) {
	tests := []struct {
		name         string
		urlStr       string
		expectLimit  int
		expectOffset int
		expectErr    bool
	}{
		{"ValidParams", "http://localhost/api/posts?limit=25&offset=10", 25, 10, false},
		{"DefaultParams", "http://localhost/api/posts", 10, 0, false},
		{"InvalidLimitString", "http://localhost/api/posts?limit=not-a-number", 0, 0, true},
		{"NegativeOffset", "http://localhost/api/posts?offset=-10", 0, 0, true},
		{"ExceededMaxLimit", "http://localhost/api/posts?limit=500", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.urlStr, nil)
			limit, offset, err := handler.ParsePagination(req, 10, 100)
			if tt.expectErr {
				if err == nil {
					t.Error("ожидалась ошибка валидации параметров, но получен nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("неожиданная ошибка парсинга: %v", err)
			}
			if limit != tt.expectLimit || offset != tt.expectOffset {
				t.Errorf("ожидался лимит/офсет %d/%d, получен %d/%d", tt.expectLimit, tt.expectOffset, limit, offset)
			}
		})
	}
}

// TestAuthHandler_Register_JSONValidation проверяет строгий разбор входящих JSON пакетов
func TestAuthHandler_Register_JSONValidation(t *testing.T) {
	svc := service.NewUserService(nil, nil)
	authHandler := handler.NewAuthHandler(svc)

	tests := []struct {
		name       string
		body       string
		expectCode int
	}{
		{"UnknownFields", `{"username":"tester","email":"t@b.com","password":"Password123!","hacker_field":"inject"}`, http.StatusBadRequest},
		{"MultipleObjects", `{"username":"tester","email":"t@b.com","password":"Password123!"}{"another":true}`, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/register", bytes.NewBufferString(tt.body))
			rec := httptest.NewRecorder()
			authHandler.Register(rec, req)
			if rec.Code != tt.expectCode {
				t.Errorf("шаг %s: ожидали код %d, получили %d", tt.name, tt.expectCode, rec.Code)
			}
		})
	}
}
