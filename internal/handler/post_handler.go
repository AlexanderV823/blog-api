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
)

type PostHandler struct {
	postService *service.PostService
}

func NewPostHandler(postService *service.PostService) *PostHandler {
	return &PostHandler{
		postService: postService,
	}
}

// Create обрабатывает создание нового поста
// POST /api/posts
// Требует аутентификации
func (h *PostHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		writeError(w, "Access token signature error", http.StatusUnauthorized)
		return
	}

	var req model.PostCreateRequest
	// Лимит 2 МБ на статью (2097152 байт) защищает от OOM/DDoS
	if err := decodeJSONStrict(w, r, &req, 2097152); err != nil {
		writeError(w, "Invalid parameters payload: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Валидация в рунах (символах кириллицы/utf-8), а не байтах
	req.Title, ok = cleanAndValidateString(req.Title, 5, 200)
	if !ok {
		writeError(w, "Title must be between 5 and 200 characters and cannot consist of spaces", http.StatusBadRequest)
		return
	}

	req.Content, ok = cleanAndValidateString(req.Content, 10, 50000)
	if !ok {
		writeError(w, "Content must be between 10 and 50000 characters and cannot consist of spaces", http.StatusBadRequest)
		return
	}

	post, err := h.postService.Create(r.Context(), userID, &req)
	if err != nil {
		// Сбой записи из-за лежащей БД превращается в 500 вместо 400
		log.Printf("[ERROR] Failed to create post due to database failure: %v", err)
		writeError(w, "Internal server error occurred", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(post)
}

// GetByID возвращает пост по ID
// GET /api/posts/{id}
// Не требует аутентификации
func (h *PostHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := extractIDFromPath(r.URL.Path, "/api/posts/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, "Invalid unique entity identity", http.StatusBadRequest)
		return
	}

	post, err := h.postService.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrPostNotFound) {
			writeError(w, err.Error(), http.StatusNotFound)
			return
		}
		// Маскируем системную ошибку при падении СУБД
		log.Printf("[ERROR] Failed to get post by ID due to database failure: %v", err)
		writeError(w, "Internal server error occurred", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(post)
}

// GetAll возвращает список постов с пагинацией
// GET /api/posts?limit=10&offset=0
// Не требует аутентификации
func (h *PostHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limit, offset, err := ParsePagination(r, 10, 100)
	if err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	posts, total, err := h.postService.GetAll(r.Context(), limit, offset)
	if err != nil {
		log.Printf("[ERROR] Failed to get posts: %v", err)
		writeError(w, "Internal server error occurred", http.StatusInternalServerError)
		return
	}

	type PaginatedResponse struct {
		Posts  []*model.Post `json:"posts"`
		Total  int           `json:"total"`
		Limit  int           `json:"limit"`
		Offset int           `json:"offset"`
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(PaginatedResponse{
		Posts:  posts,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

// Update обновляет пост
// PUT /api/posts/{id}
// Требует аутентификации, может обновить только автор
func (h *PostHandler) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		writeError(w, "Unauthorized entity validation", http.StatusUnauthorized)
		return
	}

	idStr := extractIDFromPath(r.URL.Path, "/api/posts/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, "Invalid post context identity", http.StatusBadRequest)
		return
	}

	var req model.PostUpdateRequest
	// Лимит 2 МБ на статью при обновлении (2097152 байт)
	if err := decodeJSONStrict(w, r, &req, 2097152); err != nil {
		writeError(w, "Invalid updates metadata structure: "+err.Error(), http.StatusBadRequest)
		return
	}

	req.Title, ok = cleanAndValidateString(req.Title, 5, 200)
	if !ok {
		writeError(w, "Title must be between 5 and 200 characters and cannot consist of spaces", http.StatusBadRequest)
		return
	}

	req.Content, ok = cleanAndValidateString(req.Content, 10, 50000)
	if !ok {
		writeError(w, "Content must be between 10 and 50000 characters and cannot consist of spaces", http.StatusBadRequest)
		return
	}

	post, err := h.postService.Update(r.Context(), id, userID, &req)
	if err != nil {
		if errors.Is(err, service.ErrPostNotFound) {
			writeError(w, err.Error(), http.StatusNotFound)
			return
		}
		if errors.Is(err, service.ErrForbidden) {
			writeError(w, "You cannot configure alternative entities", http.StatusForbidden)
			return
		}
		// Маскируем системную ошибку
		log.Printf("[ERROR] Failed to update post due to database failure: %v", err)
		writeError(w, "Internal server error occurred", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(post)
}

// Delete удаляет пост
// DELETE /api/posts/{id}
// Требует аутентификации, может удалить только автор
func (h *PostHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		writeError(w, "Context credentials missing", http.StatusUnauthorized)
		return
	}

	idStr := extractIDFromPath(r.URL.Path, "/api/posts/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, "Incorrect post layout key", http.StatusBadRequest)
		return
	}

	err = h.postService.Delete(r.Context(), id, userID)
	if err != nil {
		if errors.Is(err, service.ErrPostNotFound) {
			writeError(w, err.Error(), http.StatusNotFound)
			return
		}
		if errors.Is(err, service.ErrForbidden) {
			writeError(w, "Modification restricted", http.StatusForbidden)
			return
		}
		// Маскируем системную ошибку
		log.Printf("[ERROR] Failed to delete post due to database failure: %v", err)
		writeError(w, "Internal server error occurred", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetByAuthor возвращает посты конкретного автора
func (h *PostHandler) GetByAuthor(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Author selection dynamic workflow mapping", http.StatusNotImplemented)
}
