package repository

import (
	"context"
	"sync"
	"time"

	"github.com/g04/backend/internal/domain"
)

var _ domain.DashboardRepository = (*CachedDashboardRepository)(nil)

type cacheItem struct {
	data      any
	expiresAt time.Time
}

type CachedDashboardRepository struct {
	underlying domain.DashboardRepository
	ttl        time.Duration
	mu         sync.RWMutex
	cache      map[string]cacheItem
}

func NewCachedDashboardRepository(underlying domain.DashboardRepository, ttl time.Duration) *CachedDashboardRepository {
	return &CachedDashboardRepository{
		underlying: underlying,
		ttl:        ttl,
		cache:      make(map[string]cacheItem),
	}
}

func (c *CachedDashboardRepository) GetSummary(ctx context.Context) (*domain.DashboardSummary, error) {
	key := "dashboard:summary"

	c.mu.RLock()
	item, found := c.cache[key]
	if found && time.Now().Before(item.expiresAt) {
		c.mu.RUnlock()
		return item.data.(*domain.DashboardSummary), nil
	}
	c.mu.RUnlock()

	summary, err := c.underlying.GetSummary(ctx)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.cache[key] = cacheItem{
		data:      summary,
		expiresAt: time.Now().Add(c.ttl),
	}
	c.mu.Unlock()

	return summary, nil
}

func (c *CachedDashboardRepository) GetDoorTimeline(ctx context.Context, doorID int) (*domain.DoorTimeline, error) {
	return c.underlying.GetDoorTimeline(ctx, doorID)
}

func (c *CachedDashboardRepository) GetStationDoorsStatus(ctx context.Context, stationCode string) ([]*domain.DoorStatusItem, error) {
	key := "station:status:" + stationCode

	c.mu.RLock()
	item, found := c.cache[key]
	if found && time.Now().Before(item.expiresAt) {
		c.mu.RUnlock()
		return item.data.([]*domain.DoorStatusItem), nil
	}
	c.mu.RUnlock()

	status, err := c.underlying.GetStationDoorsStatus(ctx, stationCode)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.cache[key] = cacheItem{
		data:      status,
		expiresAt: time.Now().Add(c.ttl),
	}
	c.mu.Unlock()

	return status, nil
}

func (c *CachedDashboardRepository) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = make(map[string]cacheItem)
}
