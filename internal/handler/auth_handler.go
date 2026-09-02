package handler

import (
	"blog-api/internal/model"
	"blog-api/internal/service"
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

type AuthHandler struct {
	userService *service.UserService
}

func NewAuthHandler(userService *service.UserService) *AuthHandler {
	return &AuthHandler{
		userService: userService,
	}
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// Register обрабатывает запрос на регистрацию нового пользователя
// POST /api/register
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req model.UserCreateRequest
	// Защита памяти (1 МБ) и строгий разбор структуры JSON
	if err := decodeJSONStrict(w, r, &req, 1048576); err != nil {
		writeError(w, "Invalid body architecture: "+err.Error(), http.StatusBadRequest)
		return
	}

	var ok bool
	// Валидация в рунах: имя 3-50 символов (контролирует макс. длину)
	req.Username, ok = cleanAndValidateString(req.Username, 3, 50)
	if !ok {
		writeError(w, "Username must be between 3 and 50 characters and cannot consist of spaces", http.StatusBadRequest)
		return
	}

	// Валидация стандартным пакетом net/mail (разрешает верхний регистр и длинные домены)
	req.Email, ok = cleanAndValidateEmail(req.Email)
	if !ok {
		writeError(w, "Invalid email format", http.StatusBadRequest)
		return
	}

	req.Password, ok = cleanAndValidateString(req.Password, 6, 100)
	if !ok {
		writeError(w, "Password must be between 6 and 100 characters", http.StatusBadRequest)
		return
	}

	res, err := h.userService.Register(r.Context(), &req)
	if err != nil {
		// Извлекаем ошибку расширенной валидации сервиса (например, сложность пароля) через errors.As
		var valErr *service.ValidationError
		if errors.As(err, &valErr) {
			writeError(w, valErr.Error(), http.StatusBadRequest) // 400 Bad Request с текстом правила
			return
		}

		// Контролируем гонку и возвращаем предметный 409 статус
		if errors.Is(err, service.ErrUserAlreadyExists) {
			writeError(w, err.Error(), http.StatusConflict)
			return
		}
		// Перехватываем ошибки валидации сервиса, если они возникнут
		if errors.Is(err, service.ErrInvalidCredentials) {
			writeError(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Маскируем системные ошибки базы данных. Клиент видит 503 без деталей подключения.
		log.Printf("[ERROR] Critical persistence failure during registration: %v", err)
		writeError(w, "Service temporarily unavailable", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(res)
}

// Login обрабатывает запрос на вход пользователя
// POST /api/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req model.UserLoginRequest
	// Защита памяти (1 МБ) и строгий разбор структуры JSON
	if err := decodeJSONStrict(w, r, &req, 1048576); err != nil {
		writeError(w, "Invalid dynamic parameters: "+err.Error(), http.StatusBadRequest)
		return
	}

	var ok bool
	req.Email, ok = cleanAndValidateEmail(req.Email)
	if !ok {
		writeError(w, "Invalid email format", http.StatusBadRequest)
		return
	}

	req.Password, ok = cleanAndValidateString(req.Password, 1, 100)
	if !ok {
		writeError(w, "Password cannot be empty", http.StatusBadRequest)
		return
	}

	res, err := h.userService.Login(r.Context(), &req)
	if err != nil {
		var valErr *service.ValidationError
		if errors.As(err, &valErr) {
			writeError(w, valErr.Error(), http.StatusBadRequest)
			return
		}

		if errors.Is(err, service.ErrInvalidCredentials) {
			writeError(w, err.Error(), http.StatusUnauthorized)
			return
		}

		// Маскируем инфраструктурные сбои (упавший пул коннектов) в чистый 500
		log.Printf("[ERROR] Internal infrastructure failure during login: %v", err)
		writeError(w, "Internal server error occurred", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(res)
}

// GetProfile возвращает профиль текущего пользователя
func (h *AuthHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Endpoint legacy context", http.StatusNotImplemented)
}

// writeError отправляет JSON ответ с ошибкой
func writeError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}
