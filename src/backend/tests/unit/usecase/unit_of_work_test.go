package usecase_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/g04/backend/internal/adapter/repository"
	"github.com/g04/backend/internal/domain"
	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("falha ao abrir sqlite em memoria: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE inspections (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			station_code TEXT NOT NULL,
			platform TEXT NOT NULL,
			door_number INTEGER NOT NULL,
			scheduled_for TEXT,
			priority TEXT DEFAULT 'normal',
			status TEXT DEFAULT 'pending',
			notes TEXT,
			technician TEXT,
			completed_at TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		t.Fatalf("falha ao criar schema de teste: %v", err)
	}

	return db
}

func TestUnitOfWork_CommitOnSuccess(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	uow := repository.NewSQLiteUnitOfWork(db)

	err := uow.Do(context.Background(), func(store domain.UnitOfWorkStore) error {
		ins := &domain.Inspection{
			StationCode: "FAL",
			Platform:    "P1",
			DoorNumber:  4,
			Status:      "pending",
			Priority:    "critical",
		}
		return store.Inspections().Create(context.Background(), ins)
	})

	if err != nil {
		t.Fatalf("esperava sucesso na transacao: %v", err)
	}

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM inspections").Scan(&count)
	if err != nil {
		t.Fatalf("erro ao contar inspecoes: %v", err)
	}

	if count != 1 {
		t.Fatalf("esperava 1 inspecao confirmada, obteve %d", count)
	}
}

func TestUnitOfWork_RollbackOnError(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	uow := repository.NewSQLiteUnitOfWork(db)
	expectedErr := errors.New("falha simulada no meio da operacao")

	err := uow.Do(context.Background(), func(store domain.UnitOfWorkStore) error {
		ins := &domain.Inspection{
			StationCode: "PIN",
			Platform:    "P2",
			DoorNumber:  8,
			Status:      "pending",
			Priority:    "critical",
		}
		if err := store.Inspections().Create(context.Background(), ins); err != nil {
			return err
		}

		return expectedErr
	})

	if !errors.Is(err, expectedErr) {
		t.Fatalf("esperava erro %v, obteve %v", expectedErr, err)
	}

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM inspections").Scan(&count)
	if err != nil {
		t.Fatalf("erro ao contar inspecoes: %v", err)
	}

	if count != 0 {
		t.Fatalf("esperava 0 inspecoes devido ao rollback, obteve %d", count)
	}
}
