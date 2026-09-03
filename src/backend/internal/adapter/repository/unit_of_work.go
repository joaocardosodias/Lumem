package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/g04/backend/internal/domain"
)

var _ domain.UnitOfWork = (*SQLiteUnitOfWork)(nil)

type SQLiteUnitOfWork struct {
	db *sql.DB
}

func NewSQLiteUnitOfWork(db *sql.DB) *SQLiteUnitOfWork {
	return &SQLiteUnitOfWork{db: db}
}

type sqliteUnitOfWorkStore struct {
	tx *sql.Tx
}

func (s *sqliteUnitOfWorkStore) Predictions() domain.PredictionRepository {
	return NewSQLitePredictionRepository(s.tx)
}

func (s *sqliteUnitOfWorkStore) Inspections() domain.InspectionRepository {
	return NewSQLiteInspectionRepository(s.tx)
}

func (s *sqliteUnitOfWorkStore) Maintenances() domain.MaintenanceRepository {
	return NewSQLiteMaintenanceRepository(s.tx)
}

func (s *sqliteUnitOfWorkStore) Alarms() domain.AlarmRepository {
	return NewSQLiteAlarmRepository(s.tx)
}

func (u *SQLiteUnitOfWork) Do(ctx context.Context, fn func(store domain.UnitOfWorkStore) error) error {
	tx, err := u.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("falha ao iniciar transacao: %w", err)
	}

	defer func() {
		_ = tx.Rollback()
	}()

	store := &sqliteUnitOfWorkStore{tx: tx}

	if err := fn(store); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("falha ao confirmar transacao: %w", err)
	}

	return nil
}
