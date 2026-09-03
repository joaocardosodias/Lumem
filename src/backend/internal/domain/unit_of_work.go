package domain

import "context"

type UnitOfWorkStore interface {
	Predictions() PredictionRepository
	Inspections() InspectionRepository
	Maintenances() MaintenanceRepository
	Alarms() AlarmRepository
}

type UnitOfWork interface {
	Do(ctx context.Context, fn func(store UnitOfWorkStore) error) error
}
