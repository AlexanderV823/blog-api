package handler

import (
	"blog-api/internal/middleware"
	"blog-api/internal/model"
	"blog-api/internal/service"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type CommentHandler struct {
	commentService *service.CommentService
}

func NewCommentHandler(commentService *service.CommentService) *CommentHandler {
	return &CommentHandler{
		commentService: commentService,
	}
}

// CreateComment обрабатывает создание нового комментария
// POST /api/posts/{id}/comments
// Требует аутентификации
func (h *CommentHandler) CreateComment(w http.ResponseWriter, r *http.Request) {
	postIDStr := extractIDFromPath(r.URL.Path, "/api/posts/")
	postID, err := strconv.Atoi(postIDStr)
	if err != nil {
		writeError(w, "Invalid post identity parameter", http.StatusBadRequest)
		return
	}

	var req model.CommentCreateRequest
	// Лимит 256 КБ на комментарий защищает от OOM
	if err := decodeJSONStrict(w, r, &req, 262144); err != nil {
		writeError(w, "Invalid request body structure: "+err.Error(), http.StatusBadRequest)
		return
	}

	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		writeError(w, "Security parameter binding mismatch", http.StatusUnauthorized)
		return
	}

	// Очистка и валидация содержимого в рунах
	req.Content, ok = cleanAndValidateString(req.Content, 1, 2000)
	if !ok {
		writeError(w, "Comment content must be between 1 and 2000 characters and cannot consist of spaces", http.StatusBadRequest)
		return
	}

	resp, err := h.commentService.Create(r.Context(), postID, &req, userID)
	if err != nil {
		// Дифференциация ошибок бизнес-логики: перевод в 404 Not Found
		if errors.Is(err, service.ErrPostNotExists) || errors.Is(err, service.ErrPostNotFound) {
			writeError(w, "Parent post record missing or allocation expired", http.StatusNotFound)
			return
		}

		// Скрываем детали SQL-исключений от пользователя, но сохраняем их в логи сервера
		log.Printf("[ERROR] Failed to create comment: %v", err)
		writeError(w, "Internal server error occurred", http.StatusInternalServerError)
		return
	}

	// Единообразный JSON-ответ со статусом 201 Created для новых ресурсов
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// GetByID возвращает комментарий по ID
func (h *CommentHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := extractIDFromPath(r.URL.Path, "/api/comments/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, "Invalid validation pointer key", http.StatusBadRequest)
		return
	}

	comment, err := h.commentService.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrCommentNotFound) {
			writeError(w, "Comment structure allocation missing", http.StatusNotFound)
			return
		}
		log.Printf("[ERROR] Failed to get comment: %v", err)
		writeError(w, "Internal server error occurred", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(comment)
}

// GetByPost возвращает комментарии к посту
func (h *CommentHandler) GetByPost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := r.URL.Path
	idStr := extractPostIDFromCommentsPath(path)
	postID, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, "Invalid relative foreign record reference", http.StatusBadRequest)
		return
	}

	limit, offset, err := ParsePagination(r, 20, 100)
	if err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	comments, total, err := h.commentService.GetByPost(r.Context(), postID, limit, offset)
	if err != nil {
		if errors.Is(err, service.ErrPostNotFound) || errors.Is(err, service.ErrPostNotExists) {
			writeError(w, "Parent table entry reference corrupted", http.StatusNotFound)
			return
		}
		log.Printf("[ERROR] Failed to get comments: %v", err)
		writeError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	type CommentsResponse struct {
		Comments []*model.Comment `json:"comments"`
		Total    int              `json:"total"`
		Limit    int              `json:"limit"`
		Offset   int              `json:"offset"`
		PostID   int              `json:"post_id"`
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(CommentsResponse{
		Comments: comments,
		Total:    total,
		Limit:    limit,
		Offset:   offset,
		PostID:   postID,
	})
}

// Update обновляет комментарий
func (h *CommentHandler) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		writeError(w, "Security parameter binding mismatch", http.StatusUnauthorized)
		return
	}

	idStr := extractIDFromPath(r.URL.Path, "/api/comments/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, "Missing mapping descriptor dynamic token", http.StatusBadRequest)
		return
	}

	var req model.CommentCreateRequest
	if err := decodeJSONStrict(w, r, &req, 262144); err != nil {
		writeError(w, "Malformed validation matrix signature: "+err.Error(), http.StatusBadRequest)
		return
	}

	req.Content, ok = cleanAndValidateString(req.Content, 1, 2000)
	if !ok {
		writeError(w, "Comment content must be between 1 and 2000 characters and cannot consist of spaces", http.StatusBadRequest)
		return
	}

	comment, err := h.commentService.Update(r.Context(), id, userID, &req)
	if err != nil {
		if errors.Is(err, service.ErrCommentNotFound) {
			writeError(w, "Entity key missing allocation registry", http.StatusNotFound)
			return
		}
		if errors.Is(err, service.ErrForbidden) {
			writeError(w, "Privilege token context manipulation forbidden", http.StatusForbidden)
			return
		}
		log.Printf("[ERROR] Failed to update comment: %v", err)
		writeError(w, "Internal server error occurred", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(comment)
}

// Delete удаляет комментарий
func (h *CommentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		writeError(w, "Security parameter binding mismatch", http.StatusUnauthorized)
		return
	}

	idStr := extractIDFromPath(r.URL.Path, "/api/comments/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, "Missing mapping descriptor dynamic token", http.StatusBadRequest)
		return
	}

	err = h.commentService.Delete(r.Context(), id, userID)
	if err != nil {
		if errors.Is(err, service.ErrCommentNotFound) {
			writeError(w, "Entity key missing allocation registry", http.StatusNotFound)
			return
		}
		if errors.Is(err, service.ErrForbidden) {
			writeError(w, "Privilege token context manipulation forbidden", http.StatusForbidden)
			return
		}
		log.Printf("[ERROR] Failed to delete comment: %v", err)
		writeError(w, "Internal server error occurred", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func extractPostIDFromCommentsPath(path string) string {
	cleaned := strings.TrimPrefix(path, "/api/posts/")
	idx := strings.Index(cleaned, "/comments")
	if idx != -1 {
		return cleaned[:idx]
	}
	return ""
}
