package service_test

import (
	"blog-api/internal/model"
	"blog-api/internal/service"
	"context"
	"errors"
	"testing"
)

type stubCommentPostRepo struct {
	service.PostRepository
	existsResult bool
}

func (s *stubCommentPostRepo) Exists(ctx context.Context, id int) (bool, error) {
	return s.existsResult, nil
}

func TestCommentService_Create_PostNotExists(t *testing.T) {
	// Подготовка: Пост гарантированно отсутствует в базе
	postRepo := &stubCommentPostRepo{existsResult: false}

	// Вызываем оригинальный конструктор вашего CommentService
	svc := service.NewCommentService(nil, postRepo, nil)

	req := &model.CommentCreateRequest{Content: "Валидный текст комментария"}

	// Действие
	_, err := svc.Create(context.Background(), 8888, req, 1)

	// Проверка: бизнес-логика должна вернуть ErrPostNotExists
	if !errors.Is(err, service.ErrPostNotExists) {
		t.Fatalf("ожидалась бизнес-ошибка %v, получена: %v", service.ErrPostNotExists, err)
	}
}

func TestCommentService_Create_ValidationFailure(t *testing.T) {
	// Инициализируем оригинальный сервис приложения
	svc := service.NewCommentService(nil, nil, nil)
	req := &model.CommentCreateRequest{Content: ""} // Пустой контент нарушает правила рун

	// Действие
	_, err := svc.Create(context.Background(), 1, req, 1)

	// Проверка
	if err == nil {
		t.Fatal("сервисный слой пропустил пустой комментарий без ошибки валидации длины в рунах")
	}
}
