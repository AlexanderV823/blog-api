package model

import (
	"time"
)

// User представляет модель пользователя в системе
type User struct {
	ID           int       `json:"id" db:"id"`
	Username     string    `json:"username" db:"username"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"-" db:"password"` // Хешированный пароль, не отдаем в JSON
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// Post представляет модель поста в блоге
type Post struct {
	ID        int       `json:"id" db:"id"`
	Title     string    `json:"title" db:"title"`
	Content   string    `json:"content" db:"content"`
	AuthorID  int       `json:"author_id" db:"author_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Comment представляет модель комментария к посту
type Comment struct {
	ID        int       `json:"id" db:"id"`
	Content   string    `json:"content" db:"content"`
	PostID    int       `json:"post_id" db:"post_id"`
	AuthorID  int       `json:"author_id" db:"author_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// UserCreateRequest представляет запрос на создание пользователя
type UserCreateRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

// UserLoginRequest представляет запрос на вход пользователя
type UserLoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// PostCreateRequest представляет запрос на создание поста
type PostCreateRequest struct {
	Title   string `json:"title" validate:"required,min=1,max=200"`
	Content string `json:"content" validate:"required,min=1"`
}

// PostUpdateRequest представляет запрос на обновление поста
type PostUpdateRequest struct {
	Title   string `json:"title" validate:"required,min=1,max=200"`
	Content string `json:"content" validate:"required,min=1"`
}

// CommentCreateRequest представляет запрос на создание комментария
type CommentCreateRequest struct {
	Content string `json:"content" validate:"required,min=1,max=1000"`
}

// UserResponse - структура для ответа с данными пользователя (без пароля)
type UserResponse struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// TokenResponse - структура для ответа с JWT токеном
type TokenResponse struct {
	Token     string       `json:"token"`
	ExpiresAt time.Time    `json:"expires_at"`
	User      UserResponse `json:"user"`
}

// PostResponse - структура для ответа с данными поста
type PostResponse struct {
	ID        int          `json:"id"`
	Title     string       `json:"title"`
	Content   string       `json:"content"`
	Author    UserResponse `json:"author"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

// CommentResponse - структура для ответа с данными комментария
type CommentResponse struct {
	ID        int          `json:"id"`
	Content   string       `json:"content"`
	PostID    int          `json:"post_id"`
	Author    UserResponse `json:"author"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

// ToResponse преобразует User в UserResponse
func (u User) ToResponse() UserResponse {
	return UserResponse{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		CreatedAt: u.CreatedAt,
	}
}

// CanBeEditedBy проверяет, может ли пользователь редактировать пост
func (p Post) CanBeEditedBy(userID int) bool {
	return p.AuthorID == userID
}

// CanBeDeletedBy проверяет, может ли пользователь удалить пост
func (p Post) CanBeDeletedBy(userID int) bool {
	return p.AuthorID == userID
}

// CanBeEditedBy проверяет, может ли пользователь редактировать комментарий
func (c Comment) CanBeEditedBy(userID int) bool {
	return c.AuthorID == userID
}

// CanBeDeletedBy проверяет, может ли пользователь удалить комментарий
func (c Comment) CanBeDeletedBy(userID int) bool {
	return c.AuthorID == userID
}
