package service_test

import (
	"blog-api/internal/model"
	"blog-api/internal/service"
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/lib/pq"
)

// stubUserRepoFailure имитирует общие инфраструктурные сбои
type stubUserRepoFailure struct {
	service.UserRepository
	errToReturn error
}

func (s *stubUserRepoFailure) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	return nil, s.errToReturn
}

func (s *stubUserRepoFailure) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	return false, nil
}

func (s *stubUserRepoFailure) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	return false, nil
}

// stubUserRepoUniqueViolation имитирует ошибку нарушения уникальности unique_violation
type stubUserRepoUniqueViolation struct {
	service.UserRepository
}

func (s *stubUserRepoUniqueViolation) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	return false, nil
}

func (s *stubUserRepoUniqueViolation) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	return false, nil
}

func (s *stubUserRepoUniqueViolation) Create(ctx context.Context, user *model.User) error {
	pgErr := &pq.Error{
		Code: "23505", // unique_violation в PostgreSQL
	}
	// Имитируем оборачивание ошибки внутри репозитория через %w
	return fmt.Errorf("repository query insert failed: %w", pgErr)
}

// Тест: слабый пароль (без цифр) должен вызывать ValidationError
func TestUserService_Register_WeakPasswordStrength(t *testing.T) {
	svc := service.NewUserService(&stubUserRepoFailure{}, nil)

	req := &model.UserCreateRequest{
		Username: "valid_user",
		Email:    "password_strength@test.com",
		Password: "onlyletters", // Ошибка: отсутствуют цифры
	}

	_, err := svc.Register(context.Background(), req)

	var valErr *service.ValidationError
	if !errors.As(err, &valErr) {
		t.Fatalf("ожидалась ошибка типа *service.ValidationError, получена: %v", err)
	}

	if valErr.Error() != "password must contain both letters and digits" {
		t.Errorf("неверный текст ошибки валидации сложности пароля: %q", valErr.Error())
	}
}

// Тест: при критической аварии БД метод Login должен прокинуть ошибку наверх, а не маскировать её в 401
func TestUserService_Login_DatabaseInfrastructureFailure(t *testing.T) {
	dbNetworkErr := errors.New("dial tcp 127.0.0.1:5432: connect: connection refused")
	repoStub := &stubUserRepoFailure{errToReturn: dbNetworkErr}

	svc := service.NewUserService(repoStub, nil)
	req := &model.UserLoginRequest{
		Email:    "test@blog.com",
		Password: "Password123!",
	}

	_, err := svc.Login(context.Background(), req)

	// Проверяем, что вернулась именно ошибка сети, а не ErrInvalidCredentials
	if !errors.Is(err, dbNetworkErr) {
		t.Fatalf("ожидался проброс инфраструктурной ошибки %v, получен: %v", dbNetworkErr, err)
	}
}

// Тест: проверяет корректную размотку errors.As для гонки условий при регистрации
func TestUserService_Register_WrappedPqError(t *testing.T) {
	repo := &stubUserRepoUniqueViolation{}
	svc := service.NewUserService(repo, nil)

	req := &model.UserCreateRequest{
		Username: "concurrent_user",
		Email:    "race@test.com",
		Password: "Password123!",
	}

	_, err := svc.Register(context.Background(), req)

	// Если сервис использует прямое приведение вместо errors.As, этот тест упадет
	if !errors.Is(err, service.ErrUserAlreadyExists) {
		t.Fatalf("ожидался маппинг в ошибку %v, получено: %v", service.ErrUserAlreadyExists, err)
	}
}
