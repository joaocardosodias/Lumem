package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/g04/backend/internal/usecase/port"
)

var _ port.UnitOfWork = (*PostgresUnitOfWork)(nil)

type PostgresUnitOfWork struct {
	db *sql.DB
}

func NewPostgresUnitOfWork(db *sql.DB) *PostgresUnitOfWork {
	return &PostgresUnitOfWork{db: db}
}

type postgresUnitOfWorkStore struct {
	tx *sql.Tx
}

func (s *postgresUnitOfWorkStore) Predictions() port.PredictionRepository {
	return NewPostgresPredictionRepository(s.tx)
}

func (s *postgresUnitOfWorkStore) Inspections() port.InspectionRepository {
	return NewPostgresInspectionRepository(s.tx)
}

func (s *postgresUnitOfWorkStore) Maintenances() port.MaintenanceRepository {
	return NewPostgresMaintenanceRepository(s.tx)
}

func (s *postgresUnitOfWorkStore) Alarms() port.AlarmRepository {
	return NewPostgresAlarmRepository(s.tx)
}

func (s *postgresUnitOfWorkStore) Uploads() port.UploadRepository {
	return NewPostgresAlarmRepository(s.tx)
}

func (u *PostgresUnitOfWork) Do(ctx context.Context, fn func(store port.UnitOfWorkStore) error) error {
	tx, err := u.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("falha ao iniciar transacao: %w", err)
	}

	defer func() {
		_ = tx.Rollback()
	}()

	store := &postgresUnitOfWorkStore{tx: tx}

	if err := fn(store); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("falha ao confirmar transacao: %w", err)
	}

	return nil
}
