package repository

import (
	"blog-api/internal/model"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// PostRepo представляет репозиторий для работы с постами
type PostRepo struct {
	db *sql.DB
}

// NewPostRepo создает новый репозиторий постов
func NewPostRepo(db *sql.DB) *PostRepo {
	return &PostRepo{db: db}
}

// Create создает новый пост
func (r *PostRepo) Create(ctx context.Context, post *model.Post) error {
	query := `
		INSERT INTO posts (title, content, author_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`
	now := time.Now()
	post.CreatedAt = now
	post.UpdatedAt = now

	err := r.db.QueryRowContext(ctx, query, post.Title, post.Content, post.AuthorID, post.CreatedAt, post.UpdatedAt).Scan(&post.ID)
	if err != nil {
		return fmt.Errorf("failed to create post: %w", err)
	}
	return nil
}

// GetByID получает пост по ID
func (r *PostRepo) GetByID(ctx context.Context, id int) (*model.Post, error) {
	query := `
		SELECT id, title, content, author_id, created_at, updated_at
		FROM posts
		WHERE id = $1
	`
	var post model.Post
	err := r.db.QueryRowContext(ctx, query, id).Scan(&post.ID, &post.Title, &post.Content, &post.AuthorID, &post.CreatedAt, &post.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPostNotFound
		}
		return nil, fmt.Errorf("failed to get post by id: %w", err)
	}
	return &post, nil
}

// GetAll получает все посты с пагинацией
func (r *PostRepo) GetAll(ctx context.Context, limit, offset int) ([]*model.Post, error) {
	query := `
		SELECT id, title, content, author_id, created_at, updated_at
		FROM posts
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query posts: %w", err)
	}
	defer rows.Close()

	var posts []*model.Post
	for rows.Next() {
		var post model.Post
		if err := rows.Scan(&post.ID, &post.Title, &post.Content, &post.AuthorID, &post.CreatedAt, &post.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan post row: %w", err)
		}
		posts = append(posts, &post)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return posts, nil
}

// GetTotalCount получает общее количество постов
func (r *PostRepo) GetTotalCount(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM posts`
	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count posts: %w", err)
	}
	return count, nil
}

// Update обновляет пост
func (r *PostRepo) Update(ctx context.Context, post *model.Post) error {
	query := `
		UPDATE posts
		SET title = $1, content = $2, updated_at = $3
		WHERE id = $4
	`
	post.UpdatedAt = time.Now()
	res, err := r.db.ExecContext(ctx, query, post.Title, post.Content, post.UpdatedAt, post.ID)
	if err != nil {
		return fmt.Errorf("failed to update post: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrPostNotFound
	}
	return nil
}

// Delete удаляет пост
func (r *PostRepo) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM posts WHERE id = $1`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete post: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrPostNotFound
	}
	return nil
}

// Exists проверяет существование поста
func (r *PostRepo) Exists(ctx context.Context, id int) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM posts WHERE id = $1)`
	var exists bool
	err := r.db.QueryRowContext(ctx, query, id).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check post existence: %w", err)
	}
	return exists, nil
}

// GetByAuthorID получает посты определенного автора
func (r *PostRepo) GetByAuthorID(ctx context.Context, authorID int, limit, offset int) ([]*model.Post, error) {
	query := `
		SELECT id, title, content, author_id, created_at, updated_at
		FROM posts
		WHERE author_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.QueryContext(ctx, query, authorID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query author posts: %w", err)
	}
	defer rows.Close()

	var posts []*model.Post
	for rows.Next() {
		var post model.Post
		if err := rows.Scan(&post.ID, &post.Title, &post.Content, &post.AuthorID, &post.CreatedAt, &post.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan post row: %w", err)
		}
		posts = append(posts, &post)
	}

	return posts, nil
}
