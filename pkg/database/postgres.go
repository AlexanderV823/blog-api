package database

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// Внедряем SQL-файл прямо в бинарник при компиляции.
// Теперь это единственный источник истины для схемы данных СУБД.
//
//go:embed migrations/init_schema.sql
var sqlSchema string

// Migrate выполняет автоматический транзакционный накат схемы из встроенного SQL-файла
func Migrate(db *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Открываем изолированную транзакцию для безопасности схемы (DDL)
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to start migration transaction: %w", err)
	}

	// В случае паники или непредвиденного сбоя гарантируем откат изменений
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// Выполняем скрипт, прочитанный из файла init_schema.sql
	if _, err = tx.ExecContext(ctx, sqlSchema); err != nil {
		return fmt.Errorf("failed to execute migration script: %w", err)
	}

	// Фиксируем изменения в базе данных
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration transaction: %w", err)
	}

	return nil
}
