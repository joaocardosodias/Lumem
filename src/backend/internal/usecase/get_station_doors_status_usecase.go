package usecase

import (
	"context"
	"strings"

	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase/port"
)

type GetStationDoorsStatusUseCase struct {
	dashRepo port.DashboardRepository
}

func NewGetStationDoorsStatusUseCase(dashRepo port.DashboardRepository) *GetStationDoorsStatusUseCase {
	return &GetStationDoorsStatusUseCase{dashRepo: dashRepo}
}

func (uc *GetStationDoorsStatusUseCase) Execute(ctx context.Context, stationCode string) ([]DoorStatusOutputDTO, error) {
	code := strings.ToUpper(strings.TrimSpace(stationCode))
	if code == "" {
		return nil, domain.ErrStationRequired
	}

	items, err := uc.dashRepo.GetStationDoorsStatus(ctx, code)
	if err != nil {
		return nil, err
	}

	dtos := make([]DoorStatusOutputDTO, 0, len(items))
	for _, item := range items {
		dtos = append(dtos, DoorStatusOutputDTO{
			ID:         item.ID,
			Platform:   item.Platform,
			DoorNumber: item.DoorNumber,
			SAPTag:     item.SAPTag,
			Status:     item.Status,
			WillFail:   item.WillFail,
		})
	}
	return dtos, nil
}
