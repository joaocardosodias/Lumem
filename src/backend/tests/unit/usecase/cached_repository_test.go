package usecase_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/g04/backend/internal/adapter/repository"
	"github.com/g04/backend/internal/domain"
)

type countingDashboardRepo struct {
	callCount        int32
	stationCallCount int32
	timelineCalls    int32
	summaryErr       error
	stationErr       error
}

func (c *countingDashboardRepo) GetSummary(ctx context.Context) (*domain.DashboardSummary, error) {
	atomic.AddInt32(&c.callCount, 1)
	if c.summaryErr != nil {
		return nil, c.summaryErr
	}
	return &domain.DashboardSummary{TotalDoors: 528}, nil
}

func (c *countingDashboardRepo) GetDoorTimeline(ctx context.Context, doorID int) (*domain.DoorTimeline, error) {
	atomic.AddInt32(&c.timelineCalls, 1)
	return &domain.DoorTimeline{Door: domain.Door{ID: doorID}}, nil
}

func (c *countingDashboardRepo) GetStationDoorsStatus(ctx context.Context, stationCode string) ([]*domain.DoorStatusItem, error) {
	atomic.AddInt32(&c.stationCallCount, 1)
	if c.stationErr != nil {
		return nil, c.stationErr
	}
	return []*domain.DoorStatusItem{{DoorNumber: 1}}, nil
}

func TestCachedDashboardRepository_CacheHitAndInvalidate(t *testing.T) {
	underlying := &countingDashboardRepo{}
	cached := repository.NewCachedDashboardRepository(underlying, 100*time.Millisecond)

	s1, err := cached.GetSummary(context.Background())
	if err != nil || s1.TotalDoors != 528 {
		t.Fatalf("esperava 528 portas")
	}
	if atomic.LoadInt32(&underlying.callCount) != 1 {
		t.Fatalf("deveria ter chamado o banco 1 vez")
	}

	s2, err := cached.GetSummary(context.Background())
	if err != nil || s2.TotalDoors != 528 {
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

func TestCachedDashboardRepository_StationDoorsStatusCacheHit(t *testing.T) {
	underlying := &countingDashboardRepo{}
	cached := repository.NewCachedDashboardRepository(underlying, 100*time.Millisecond)

	items1, err := cached.GetStationDoorsStatus(context.Background(), "FAL")
	if err != nil || len(items1) != 1 {
		t.Fatalf("esperava 1 item")
	}
	if atomic.LoadInt32(&underlying.stationCallCount) != 1 {
		t.Fatalf("esperava 1 chamada ao repositorio base")
	}

	items2, err := cached.GetStationDoorsStatus(context.Background(), "FAL")
	if err != nil || len(items2) != 1 {
		t.Fatalf("esperava 1 item servido do cache")
	}
	if atomic.LoadInt32(&underlying.stationCallCount) != 1 {
		t.Fatalf("segunda chamada deveria ter sido servida pelo cache")
	}

	cached.Invalidate()
	_, _ = cached.GetStationDoorsStatus(context.Background(), "FAL")
	if atomic.LoadInt32(&underlying.stationCallCount) != 2 {
		t.Fatalf("esperava nova chamada apos invalidacao")
	}
}

func TestCachedDashboardRepository_GetDoorTimeline(t *testing.T) {
	underlying := &countingDashboardRepo{}
	cached := repository.NewCachedDashboardRepository(underlying, 100*time.Millisecond)

	tl, err := cached.GetDoorTimeline(context.Background(), 42)
	if err != nil || tl.Door.ID != 42 {
		t.Fatalf("esperava repasse correto para underlying")
	}
	if atomic.LoadInt32(&underlying.timelineCalls) != 1 {
		t.Fatalf("esperava 1 chamada ao repositorio base")
	}
}

func TestCachedDashboardRepository_Errors(t *testing.T) {
	underlying := &countingDashboardRepo{
		summaryErr: errors.New("falha summary"),
		stationErr: errors.New("falha station"),
	}
	cached := repository.NewCachedDashboardRepository(underlying, 100*time.Millisecond)

	_, err := cached.GetSummary(context.Background())
	if err == nil || err.Error() != "falha summary" {
		t.Fatalf("esperava erro de summary")
	}

	_, err = cached.GetStationDoorsStatus(context.Background(), "FAL")
	if err == nil || err.Error() != "falha station" {
		t.Fatalf("esperava erro de station")
	}
}
