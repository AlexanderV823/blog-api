package auth_test

import (
	"blog-api/pkg/auth"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestJWTManager_GenerateAndValidate(t *testing.T) {
	secret := "super-secret-key-32-chars-long!!"
	manager := auth.NewJWTManager(secret, 1)

	// 1. Тест успешной генерации и валидации
	tokenStr, expiry, err := manager.GenerateToken(42, "test@blog.com", "blogger")
	if err != nil {
		t.Fatalf("не удалось сгенерировать токен: %v", err)
	}

	if expiry.Before(time.Now()) {
		t.Error("время жизни токена создано в прошлом")
	}

	claims, err := manager.ValidateToken(tokenStr)
	if err != nil {
		t.Fatalf("валидный токен отклонен менеджером: %v", err)
	}

	if claims.UserID != 42 || claims.Email != "test@blog.com" || claims.Username != "blogger" {
		t.Errorf("данные claims повредились при распаковке, получено: %+v", claims)
	}
}

func TestJWTManager_ExpiredToken(t *testing.T) {
	// Инициализируем менеджер с нулевым временем жизни для симуляции просрочки
	secret := "super-secret-key-32-chars-long!!"
	manager := auth.NewJWTManager(secret, 0)

	tokenStr, _, _ := manager.GenerateToken(1, "user@test.com", "user")

	// Искусственно ждем 1 миллисекунду, чтобы токен гарантированно устарел
	time.Sleep(1 * time.Millisecond)

	_, err := manager.ValidateToken(tokenStr)
	if !errors.Is(err, auth.ErrExpiredToken) {
		t.Errorf("ожидалась ошибка %v для просроченного токена, получена: %v", auth.ErrExpiredToken, err)
	}
}

func TestJWTManager_InvalidSignatureAndAlgNone(t *testing.T) {
	secretA := "secret-key-number-one-32-chars!!"
	secretB := "secret-key-number-two-32-chars!!"

	managerA := auth.NewJWTManager(secretA, 1)
	managerB := auth.NewJWTManager(secretB, 1)

	// Токен, подписанный ключом А, должен быть отклонен менеджером Б
	tokenStrA, _, _ := managerA.GenerateToken(10, "a@test.com", "user_a")
	_, err := managerB.ValidateToken(tokenStrA)
	if !errors.Is(err, auth.ErrInvalidToken) {
		t.Errorf("токен с чужой подписью должен возвращать %v, получено: %v", auth.ErrInvalidToken, err)
	}

	// Симуляция атаки с подменой алгоритма на "none" (неподписанный токен)
	unsafeClaims := auth.Claims{
		UserID:   999,
		Email:    "hacker@test.com",
		Username: "attacker",
	}
	unsignedToken := jwt.NewWithClaims(jwt.SigningMethodNone, unsafeClaims)
	unsafeTokenStr, _ := unsignedToken.SignedString(jwt.UnsafeAllowNoneSignatureType)

	_, err = managerA.ValidateToken(unsafeTokenStr)
	if err == nil {
		t.Error("критическая уязвимость подписи: менеджер принял неподписанный токен с alg:none")
	}
}
