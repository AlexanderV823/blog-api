package service_test

import (
	"blog-api/internal/model"
	"blog-api/internal/service"
	"context"
	"strings"
	"testing"
)

type stubPostRepo struct {
	service.PostRepository
	savedPost *model.Post
}

func (s *stubPostRepo) Create(ctx context.Context, post *model.Post) error {
	s.savedPost = post
	return nil
}

func TestPostService_Create_ValidPost(t *testing.T) {
	// Подготовка
	repo := &stubPostRepo{}
	svc := service.NewPostService(repo, nil)

	req := &model.PostCreateRequest{
		Title:   "Мой заголовок",
		Content: "Текст статьи на кириллице",
	}

	// Действие
	res, err := svc.Create(context.Background(), 42, req)

	// Проверка
	if err != nil {
		t.Fatalf("ожидалось успешное создание поста, получена ошибка: %v", err)
	}
	if repo.savedPost == nil {
		t.Fatal("репозиторий создания поста не был вызван")
	}
	if repo.savedPost.AuthorID != 42 {
		t.Errorf("ожидался AuthorID = 42, получен: %d", repo.savedPost.AuthorID)
	}
	if res.Title != req.Title {
		t.Errorf("ожидался заголовок %q, получен: %q", req.Title, res.Title)
	}
}

func TestPostService_Create_TitleTooLong(t *testing.T) {
	// Подготовка: генерируем заголовок длиной 201 руна
	svc := service.NewPostService(nil, nil)
	longTitle := strings.Repeat("а", 201)

	req := &model.PostCreateRequest{
		Title:   longTitle,
		Content: "Валидный контент",
	}

	// Действие
	_, err := svc.Create(context.Background(), 1, req)

	// Проверка
	if err == nil {
		t.Fatal("ожидался провал валидации из-за превышения длины заголовка в рунах")
	}
}
