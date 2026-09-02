package service

import (
	"blog-api/internal/model"
	"blog-api/internal/repository"
	"context"
	"errors"
	"fmt"
	"unicode/utf8"
)

var (
	ErrPostNotFound = errors.New("post not found")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
)

type PostService struct {
	postRepo PostRepository
	userRepo UserRepository
}

func NewPostService(postRepo PostRepository, userRepo UserRepository) *PostService {
	return &PostService{
		postRepo: postRepo,
		userRepo: userRepo,
	}
}

func (s *PostService) Create(ctx context.Context, userID int, req *model.PostCreateRequest) (*model.Post, error) {
	if err := validatePostCreateRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	post := &model.Post{
		Title:    req.Title,
		Content:  req.Content,
		AuthorID: userID,
	}

	if err := s.postRepo.Create(ctx, post); err != nil {
		return nil, err
	}

	return post, nil
}

func (s *PostService) GetByID(ctx context.Context, id int) (*model.Post, error) {
	post, err := s.postRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrPostNotFound) {
			return nil, ErrPostNotFound
		}
		return nil, err
	}
	return post, nil
}

func (s *PostService) GetAll(ctx context.Context, limit, offset int) ([]*model.Post, int, error) {
	if limit <= 0 {
		limit = 10
	} else if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	posts, err := s.postRepo.GetAll(ctx, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.postRepo.GetTotalCount(ctx)
	if err != nil {
		return nil, 0, err
	}

	return posts, total, nil
}

func (s *PostService) Update(ctx context.Context, id int, userID int, req *model.PostUpdateRequest) (*model.Post, error) {
	post, err := s.postRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrPostNotFound) {
			return nil, ErrPostNotFound
		}
		return nil, err
	}

	if post.AuthorID != userID {
		return nil, ErrForbidden
	}

	if err := validatePostUpdateRequest(req); err != nil {
		return nil, err
	}

	if req.Title != "" {
		post.Title = req.Title
	}
	if req.Content != "" {
		post.Content = req.Content
	}

	if err := s.postRepo.Update(ctx, post); err != nil {
		return nil, err
	}

	return post, nil
}

func (s *PostService) Delete(ctx context.Context, id int, userID int) error {
	post, err := s.postRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrPostNotFound) {
			return ErrPostNotFound
		}
		return err
	}

	if post.AuthorID != userID {
		return ErrForbidden
	}

	return s.postRepo.Delete(ctx, id)
}

func (s *PostService) GetByAuthor(ctx context.Context, authorID int, limit, offset int) ([]*model.Post, int, error) {
	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	posts, err := s.postRepo.GetByAuthorID(ctx, authorID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	return posts, len(posts), nil
}

func validatePostCreateRequest(req *model.PostCreateRequest) error {
	if req.Title == "" || utf8.RuneCountInString(req.Title) > 200 {
		return errors.New("title must be present and cannot exceed 200 characters")
	}
	if req.Content == "" || utf8.RuneCountInString(req.Content) > 50000 {
		return errors.New("content cannot be empty and cannot exceed 50000 characters")
	}
	return nil
}

// validatePostUpdateRequest проверяет корректность данных для обновления поста
func validatePostUpdateRequest(req *model.PostUpdateRequest) error {
	if req.Title != "" && utf8.RuneCountInString(req.Title) > 200 {
		return errors.New("title cannot exceed 200 characters")
	}
	if req.Content != "" && utf8.RuneCountInString(req.Content) > 50000 {
		return errors.New("content cannot exceed 50000 characters")
	}
	return nil
}
