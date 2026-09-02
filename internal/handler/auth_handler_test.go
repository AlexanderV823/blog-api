package handler_test

import (
	"blog-api/internal/handler"
	"blog-api/internal/service"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockUserRepoForHandler struct {
	service.UserRepository
	dbError error
}

func (m *mockUserRepoForHandler) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	return false, m.dbError
}

func TestAuthHandler_Register_InfrastructureFailureResponse(t *testing.T) {
	// Имитируем падение базы данных (ошибка подключения)
	dbErr := errors.New("sql: database is closed or connection timed out")
	repoMock := &mockUserRepoForHandler{dbError: dbErr}

	svc := service.NewUserService(repoMock, nil)
	authHandler := handler.NewAuthHandler(svc)

	reqBody := `{"username":"tester","email":"test@blog.com","password":"Password123!"}`
	req := httptest.NewRequest(http.MethodPost, "/api/register", bytes.NewBufferString(reqBody))
	rec := httptest.NewRecorder()

	// Вызываем оригинальный метод хендлера регистрации
	authHandler.Register(rec, req)

	// Проверяем, что инфраструктурный сбой возвращает строго 503 Service Unavailable
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("ожидался статус 503 для падения БД, получен: %d", rec.Code)
	}

	var jsonResp map[string]string
	_ = json.Unmarshal(rec.Body.Bytes(), &jsonResp)

	// Убеждаемся, что внутренняя ошибка скрыта от клиента согласно контракту безопасности
	if jsonResp["error"] != "Service temporarily unavailable" {
		t.Errorf("детали СУБД просочились наружу, получен JSON: %q", rec.Body.String())
	}
}
