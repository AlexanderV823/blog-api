package service

import (
	"blog-api/internal/model"
	"blog-api/internal/repository"
	"blog-api/pkg/auth"
	"context"
	"errors"
	"net/mail"
	"strings"
	"unicode/utf8"

	"github.com/lib/pq"
)

// ValidationError представляет ошибку валидации данных на уровне бизнес-логики
type ValidationError struct {
	Err error
}

func (e *ValidationError) Error() string {
	return e.Err.Error()
}

func (e *ValidationError) Unwrap() error {
	return e.Err
}

var (
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotFound       = errors.New("user not found")
)

type UserService struct {
	userRepo   UserRepository
	jwtManager *auth.JWTManager
}

func NewUserService(userRepo UserRepository, jwtManager *auth.JWTManager) *UserService {
	return &UserService{
		userRepo:   userRepo,
		jwtManager: jwtManager,
	}
}

func (s *UserService) Register(ctx context.Context, req *model.UserCreateRequest) (*model.TokenResponse, error) {
	if err := validateUserCreateRequest(req); err != nil {
		// Оборачиваем ошибку валидации (включая проверку сложности пароля) в типизированную структуру
		return nil, &ValidationError{Err: err}
	}

	emailExists, err := s.userRepo.ExistsByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if emailExists {
		return nil, ErrUserAlreadyExists
	}

	userExists, err := s.userRepo.ExistsByUsername(ctx, req.Username)
	if err != nil {
		return nil, err
	}
	if userExists {
		return nil, ErrUserAlreadyExists
	}

	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: passwordHash,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		var pgErr *pq.Error
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" { // Код unique_violation в PostgreSQL
				return nil, ErrUserAlreadyExists
			}
		}
		return nil, err
	}

	token, expiresAt, err := s.jwtManager.GenerateToken(user.ID, user.Email, user.Username)
	if err != nil {
		return nil, err
	}

	return &model.TokenResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      user.ToResponse(),
	}, nil
}

func (s *UserService) Login(ctx context.Context, req *model.UserLoginRequest) (*model.TokenResponse, error) {
	if err := validateUserLoginRequest(req); err != nil {
		return nil, &ValidationError{Err: err}
	}

	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		// Превращаем в ErrInvalidCredentials ТОЛЬКО если пользователь действительно отсутствует в БД
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		// Инфраструктурные ошибки (connection refused, timeout СУБД) прокидываем выше для маскирования в 500/503
		return nil, err
	}

	if !auth.CheckPassword(req.Password, user.PasswordHash) {
		return nil, ErrInvalidCredentials
	}

	token, expiresAt, err := s.jwtManager.GenerateToken(user.ID, user.Email, user.Username)
	if err != nil {
		return nil, err
	}

	return &model.TokenResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      user.ToResponse(),
	}, nil
}

func (s *UserService) GetByID(ctx context.Context, id int) (*model.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

func (s *UserService) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	return s.userRepo.GetByEmail(ctx, email)
}

// validateUserCreateRequest проверяет корректность данных для регистрации
func validateUserCreateRequest(req *model.UserCreateRequest) error {
	if utf8.RuneCountInString(req.Username) < 3 || utf8.RuneCountInString(req.Username) > 50 {
		return errors.New("username must be between 3 and 50 characters long")
	}

	// Стандартная проверка структуры адреса, игнорирующая регистр и ограничения зон верхнего уровня
	cleanedEmail := strings.ToLower(strings.TrimSpace(req.Email))
	if _, err := mail.ParseAddress(cleanedEmail); err != nil {
		return errors.New("invalid email format")
	}

	return auth.ValidatePasswordStrength(req.Password)
}

// validateUserLoginRequest проверяет корректность данных для входа
func validateUserLoginRequest(req *model.UserLoginRequest) error {
	if req.Email == "" || req.Password == "" {
		return errors.New("email and password are required")
	}
	return nil
}
