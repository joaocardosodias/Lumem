package usecase_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/g04/backend/internal/adapter/repository"
	"github.com/g04/backend/internal/domain"
)

type countingDashboardRepo struct {
	callCount int32
}

func (c *countingDashboardRepo) GetSummary(ctx context.Context) (*domain.DashboardSummary, error) {
	atomic.AddInt32(&c.callCount, 1)
	return &domain.DashboardSummary{TotalDoors: 528}, nil
}

func (c *countingDashboardRepo) GetDoorTimeline(ctx context.Context, doorID int) (*domain.DoorTimeline, error) {
	return nil, nil
}

func (c *countingDashboardRepo) GetStationDoorsStatus(ctx context.Context, stationCode string) ([]*domain.DoorStatusItem, error) {
	atomic.AddInt32(&c.callCount, 1)
	return []*domain.DoorStatusItem{{DoorNumber: 1}}, nil
}

func TestCachedDashboardRepository_CacheHitAndInvalidate(t *testing.T) {
	underlying := &countingDashboardRepo{}
	cached := repository.NewCachedDashboardRepository(underlying, 100*time.Millisecond)

	s1, _ := cached.GetSummary(context.Background())
	if s1.TotalDoors != 528 {
		t.Fatalf("esperava 528 portas")
	}
	if atomic.LoadInt32(&underlying.callCount) != 1 {
		t.Fatalf("deveria ter chamado o banco 1 vez")
	}

	s2, _ := cached.GetSummary(context.Background())
	if s2.TotalDoors != 528 {
		t.Fatalf("esperava 528 portas")
	}
	if atomic.LoadInt32(&underlying.callCount) != 1 {
		t.Fatalf("deveria ter servido do cache sem nova chamada ao banco")
	}

	cached.Invalidate()

	_, _ = cached.GetSummary(context.Background())
	if atomic.LoadInt32(&underlying.callCount) != 2 {
		t.Fatalf("apos invalidacao deveria ter chamado o banco novamente")
	}
}
