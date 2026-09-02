package repository

import (
	"blog-api/internal/model"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// CommentRepo представляет репозиторий для работы с комментариями
type CommentRepo struct {
	db *sql.DB
}

// NewCommentRepo создает новый репозиторий комментариев
func NewCommentRepo(db *sql.DB) *CommentRepo {
	return &CommentRepo{db: db}
}

// Create создает новый комментарий
func (r *CommentRepo) Create(ctx context.Context, comment *model.Comment) error {
	query := `
		INSERT INTO comments (content, post_id, author_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`
	now := time.Now()
	comment.CreatedAt = now
	comment.UpdatedAt = now

	err := r.db.QueryRowContext(ctx, query, comment.Content, comment.PostID, comment.AuthorID, comment.CreatedAt, comment.UpdatedAt).Scan(&comment.ID)
	if err != nil {
		return fmt.Errorf("failed to create comment: %w", err)
	}
	return nil
}

// GetByID получает комментарий по ID
func (r *CommentRepo) GetByID(ctx context.Context, id int) (*model.Comment, error) {
	query := `
		SELECT id, content, post_id, author_id, created_at, updated_at
		FROM comments
		WHERE id = $1
	`
	var comment model.Comment
	err := r.db.QueryRowContext(ctx, query, id).Scan(&comment.ID, &comment.Content, &comment.PostID, &comment.AuthorID, &comment.CreatedAt, &comment.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCommentNotFound
		}
		return nil, fmt.Errorf("failed to get comment: %w", err)
	}
	return &comment, nil
}

// GetByPostID получает комментарии к посту с пагинацией
func (r *CommentRepo) GetByPostID(ctx context.Context, postID int, limit, offset int) ([]*model.Comment, error) {
	query := `
		SELECT id, content, post_id, author_id, created_at, updated_at
		FROM comments
		WHERE post_id = $1
		ORDER BY created_at ASC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.QueryContext(ctx, query, postID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get comments: %w", err)
	}
	defer rows.Close()

	var comments []*model.Comment
	for rows.Next() {
		var comment model.Comment
		err := rows.Scan(&comment.ID, &comment.Content, &comment.PostID, &comment.AuthorID, &comment.CreatedAt, &comment.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan comment: %w", err)
		}
		comments = append(comments, &comment)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate comments: %w", err)
	}

	return comments, nil
}

// GetCountByPostID получает количество комментариев к посту
func (r *CommentRepo) GetCountByPostID(ctx context.Context, postID int) (int, error) {
	query := `SELECT COUNT(*) FROM comments WHERE post_id = $1`
	var count int
	err := r.db.QueryRowContext(ctx, query, postID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get comments count: %w", err)
	}
	return count, nil
}

// Update обновляет комментарий
func (r *CommentRepo) Update(ctx context.Context, comment *model.Comment) error {
	query := `
		UPDATE comments
		SET content = $1, updated_at = $2
		WHERE id = $3
	`
	comment.UpdatedAt = time.Now()
	res, err := r.db.ExecContext(ctx, query, comment.Content, comment.UpdatedAt, comment.ID)
	if err != nil {
		return fmt.Errorf("failed to update comment: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrCommentNotFound
	}
	return nil
}

// Delete удаляет комментарий
func (r *CommentRepo) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM comments WHERE id = $1`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete comment: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrCommentNotFound
	}
	return nil
}
