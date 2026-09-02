package repository_test

import (
	"blog-api/internal/repository"
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestUserRepo_GetByID_SuccessAndNotFound(t *testing.T) {
	// Создаем виртуальное SQL соединение и мок-объект ожиданий
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("не удалось запустить sqlmock: %v", err)
	}
	defer db.Close()

	repo := repository.NewUserRepo(db)
	now := time.Now() // Используем реальный объект времени для Scan

	// Сценарий 1: Успешный выбор пользователя
	rows := sqlmock.NewRows([]string{"id", "username", "email", "password_hash", "created_at", "updated_at"}).
		AddRow(1, "ivan", "ivan@test.com", "hash123", now, now)

	// Эскейпим регулярное выражение запроса, так как репозиторий выполняет SELECT ... WHERE id = $1
	mock.ExpectQuery(`SELECT id, username, email, password_hash, created_at, updated_at FROM users WHERE id = \$1`).
		WithArgs(1).
		WillReturnRows(rows)

	user, err := repo.GetByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("ожидалось успешное чтение из репозитория, получена ошибка: %v", err)
	}
	if user.Username != "ivan" {
		t.Errorf("ожидалось имя 'ivan', отсканировано: %q", user.Username)
	}

	// Сценарий 2: Обработка sql.ErrNoRows (Пользователь не найден)
	mock.ExpectQuery(`SELECT id, username, email, password_hash, created_at, updated_at FROM users WHERE id = \$1`).
		WithArgs(999).
		WillReturnError(sql.ErrNoRows)

	_, err = repo.GetByID(context.Background(), 999)
	if !errors.Is(err, repository.ErrUserNotFound) {
		t.Errorf("ожидался маппинг в ошибку repository.ErrUserNotFound, получено: %v", err)
	}
}

func TestPostRepo_Exists(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("не удалось запустить sqlmock: %v", err)
	}
	defer db.Close()

	repo := repository.NewPostRepo(db)

	rows := sqlmock.NewRows([]string{"exists"}).AddRow(true)
	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM posts WHERE id = \$1\)`).
		WithArgs(42).
		WillReturnRows(rows)

	exists, err := repo.Exists(context.Background(), 42)
	if err != nil || !exists {
		t.Errorf("ожидалось true для существующего поста, получено: %v (exists: %t)", err, exists)
	}
}
