package auth

import (
	"crypto/rand"
	"errors"
	"math/big"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmptyPassword    = errors.New("password cannot be empty")
	ErrPasswordTooShort = errors.New("password is too short")
)

// HashPassword хеширует пароль используя bcrypt
func HashPassword(password string) (string, error) {
	if password == "" {
		return "", ErrEmptyPassword
	}

	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// CheckPassword проверяет соответствие пароля и его хеша
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// ValidatePasswordStrength проверяет надежность пароля
func ValidatePasswordStrength(password string) error {
	if len(password) < 6 {
		return ErrPasswordTooShort
	}

	var hasLetter, hasDigit bool
	for _, r := range password {
		if unicode.IsLetter(r) {
			hasLetter = true
		}
		if unicode.IsDigit(r) {
			hasDigit = true
		}
	}

	if !hasLetter || !hasDigit {
		return errors.New("password must contain both letters and digits")
	}

	return nil
}

// GenerateRandomPassword генерирует случайный пароль (опциональное задание)
func GenerateRandomPassword(length int) (string, error) {
	if length < 1 {
		return "", errors.New("invalid length")
	}

	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	result := make([]byte, length)

	for i := range result {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		result[i] = charset[num.Int64()]
	}

	return string(result), nil
}
