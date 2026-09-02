package service

import (
	"blog-api/internal/model"
	"context"
)

// UserRepository описывает контракт для работы с пользователями
type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	GetByID(ctx context.Context, id int) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	ExistsByUsername(ctx context.Context, username string) (bool, error)
}

// PostRepository описывает контракт для работы с публикациями
type PostRepository interface {
	Create(ctx context.Context, post *model.Post) error
	GetByID(ctx context.Context, id int) (*model.Post, error)
	GetAll(ctx context.Context, limit, offset int) ([]*model.Post, error)
	GetTotalCount(ctx context.Context) (int, error)
	Update(ctx context.Context, post *model.Post) error
	Delete(ctx context.Context, id int) error
	GetByAuthorID(ctx context.Context, authorID int, limit, offset int) ([]*model.Post, error)
	Exists(ctx context.Context, id int) (bool, error)
}

// CommentRepository описывает контракт для работы с комментариями
type CommentRepository interface {
	Create(ctx context.Context, comment *model.Comment) error
	GetByID(ctx context.Context, id int) (*model.Comment, error)
	GetByPostID(ctx context.Context, postID int, limit, offset int) ([]*model.Comment, error)
	GetCountByPostID(ctx context.Context, postID int) (int, error)
	Update(ctx context.Context, comment *model.Comment) error
	Delete(ctx context.Context, id int) error
}
