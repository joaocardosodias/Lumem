package usecase

import (
	"context"

	"github.com/g04/backend/internal/usecase/port"
)

type GetDashboardSummaryUseCase struct {
	dashRepo port.DashboardRepository
}

func NewGetDashboardSummaryUseCase(dashRepo port.DashboardRepository) *GetDashboardSummaryUseCase {
	return &GetDashboardSummaryUseCase{dashRepo: dashRepo}
}

func (uc *GetDashboardSummaryUseCase) Execute(ctx context.Context) (*DashboardSummaryOutputDTO, error) {
	s, err := uc.dashRepo.GetSummary(ctx)
	if err != nil {
		return nil, err
	}

	critical := make([]StationRiskOutputDTO, 0, len(s.CriticalStations))
	for _, c := range s.CriticalStations {
		critical = append(critical, StationRiskOutputDTO{
			StationCode: c.StationCode,
			DoorsAtRisk: c.DoorsAtRisk,
		})
	}

	return &DashboardSummaryOutputDTO{
		TotalDoors:           s.TotalDoors,
		DoorsAtRisk:          s.DoorsAtRisk,
		PendingInspections:   s.PendingInspections,
		CompletedInspections: s.CompletedInspections,
		CriticalStations:     critical,
	}, nil
}
