package service

import (
	"blog-api/internal/model"
	"blog-api/internal/repository"
	"context"
	"errors"
	"fmt"
	"time"
	"unicode/utf8"
)

var (
	ErrCommentNotFound = errors.New("comment not found")
	ErrPostNotExists   = errors.New("post does not exist")
)

type CommentService struct {
	commentRepo CommentRepository
	postRepo    PostRepository
	userRepo    UserRepository
}

func NewCommentService(commentRepo CommentRepository, postRepo PostRepository, userRepo UserRepository) *CommentService {
	return &CommentService{
		commentRepo: commentRepo,
		postRepo:    postRepo,
		userRepo:    userRepo,
	}
}

func (s *CommentService) Create(ctx context.Context, postID int, req *model.CommentCreateRequest, authorID int) (*model.CommentResponse, error) {
	// Валидация текста комментария в рунах
	if err := validateCommentCreateRequest(req); err != nil {
		return nil, err
	}

	postExists, err := s.postRepo.Exists(ctx, postID)
	if err != nil {
		return nil, err
	}
	if !postExists {
		return nil, ErrPostNotExists
	}

	comment := &model.Comment{
		Content:   req.Content,
		PostID:    postID,
		AuthorID:  authorID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.commentRepo.Create(ctx, comment); err != nil {
		return nil, err
	}

	author, err := s.userRepo.GetByID(ctx, authorID)
	if err != nil {
		return nil, err
	}

	return &model.CommentResponse{
		ID:        comment.ID,
		Content:   comment.Content,
		PostID:    comment.PostID,
		Author:    author.ToResponse(),
		CreatedAt: comment.CreatedAt,
		UpdatedAt: comment.UpdatedAt,
	}, nil
}

func (s *CommentService) GetByID(ctx context.Context, id int) (*model.Comment, error) {
	comment, err := s.commentRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrCommentNotFound) {
			return nil, ErrCommentNotFound
		}
		return nil, err
	}
	return comment, nil
}

func (s *CommentService) GetByPost(ctx context.Context, postID int, limit, offset int) ([]*model.Comment, int, error) {
	if limit <= 0 {
		limit = 20
	} else if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	postExists, err := s.postRepo.Exists(ctx, postID)
	if err != nil {
		return nil, 0, err
	}
	if !postExists {
		return nil, 0, ErrPostNotExists
	}

	comments, err := s.commentRepo.GetByPostID(ctx, postID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.commentRepo.GetCountByPostID(ctx, postID)
	if err != nil {
		return nil, 0, err
	}

	return comments, total, nil
}

func (s *CommentService) Update(ctx context.Context, id int, userID int, req *model.CommentCreateRequest) (*model.Comment, error) {
	comment, err := s.commentRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrCommentNotFound) {
			return nil, ErrCommentNotFound
		}
		return nil, err
	}

	if comment.AuthorID != userID {
		return nil, ErrForbidden
	}

	if err := validateCommentUpdateRequest(req); err != nil {
		return nil, err
	}

	comment.Content = req.Content

	if err := s.commentRepo.Update(ctx, comment); err != nil {
		return nil, err
	}

	return comment, nil
}

func (s *CommentService) Delete(ctx context.Context, id int, userID int) error {
	comment, err := s.commentRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrCommentNotFound) {
			return ErrCommentNotFound
		}
		return err
	}

	if comment.AuthorID != userID {
		return ErrForbidden
	}

	return s.commentRepo.Delete(ctx, id)
}

func (s *CommentService) GetByAuthor(ctx context.Context, authorID int, limit, offset int) ([]*model.Comment, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	return nil, 0, fmt.Errorf("not implemented")
}

// validateCommentCreateRequest проверяет корректность данных для создания комментария
func validateCommentCreateRequest(req *model.CommentCreateRequest) error {
	if req.Content == "" || utf8.RuneCountInString(req.Content) > 2000 {
		return errors.New("content must be present and cannot exceed 2000 characters")
	}
	return nil
}

// validateCommentUpdateRequest проверяет корректность данных для обновления комментария
func validateCommentUpdateRequest(req *model.CommentCreateRequest) error {
	if req.Content == "" || utf8.RuneCountInString(req.Content) > 2000 {
		return errors.New("content must be present and cannot exceed 2000 characters")
	}
	return nil
}
