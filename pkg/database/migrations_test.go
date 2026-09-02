package database_test

import (
	"blog-api/pkg/database"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestMigrate_RealTransaction_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("ошибка инициализации sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS users").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	err = database.Migrate(db)
	if err != nil {
		t.Fatalf("настоящая миграция упала со сбоем: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("не все ожидания транзакции были выполнены: %v", err)
	}
}

func TestMigrate_RealTransaction_Rollback(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("ошибка инициализации sqlmock: %v", err)
	}
	defer db.Close()

	// Имитируем сбой СУБД при выполнении монолитного скрипта
	mock.ExpectBegin()
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS users").
		WillReturnError(errors.New("postgres: connection lost or syntax error"))
	mock.ExpectRollback()

	err = database.Migrate(db)
	if err == nil {
		t.Error("мигратор пропустил ошибку СУБД и не вызвал Rollback транзакции")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("транзакция принудительного отката повреждена: %v", err)
	}
}
