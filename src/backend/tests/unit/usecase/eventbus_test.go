package usecase_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/g04/backend/internal/adapter/eventbus"
	"github.com/g04/backend/internal/domain"
	"github.com/g04/backend/internal/usecase"
	"github.com/g04/backend/internal/usecase/port"
)

type mockCacheInvalidator struct {
	invalidated bool
}

func (m *mockCacheInvalidator) Invalidate() {
	m.invalidated = true
}

type mockEventBus struct {
	mu        sync.Mutex
	published []domain.Event
}

func (m *mockEventBus) Publish(ctx context.Context, event domain.Event) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.published = append(m.published, event)
}

func (m *mockEventBus) PublishSync(ctx context.Context, event domain.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.published = append(m.published, event)
	return nil
}

func (m *mockEventBus) Subscribe(eventName string, handler port.EventHandler) {}

func TestEventBus_PublishAndSubscribe(t *testing.T) {
	bus := eventbus.NewInMemoryEventBus(nil)
	received := make(chan string, 1)

	bus.Subscribe("DoorRiskDetected", func(ctx context.Context, e domain.Event) error {
		evt := e.(domain.DoorRiskDetectedEvent)
		received <- evt.StationCode
		return nil
	})

	bus.Publish(context.Background(), domain.DoorRiskDetectedEvent{
		RunID:       "run-123",
		StationCode: "FAL",
		DoorNumber:  4,
		Timestamp:   time.Now(),
	})

	select {
	case station := <-received:
		if station != "FAL" {
			t.Fatalf("esperava estacao FAL, obteve %s", station)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("timeout aguardando evento no subscriber")
	}
}

func TestSubscribers_AutoInspectionOnDoorRisk(t *testing.T) {
	bus := eventbus.NewInMemoryEventBus(nil)
	insRepo := newMockInspectionRepo()
	cache := &mockCacheInvalidator{}

	usecase.RegisterEventSubscribers(bus, insRepo, cache, nil)

	err := bus.PublishSync(context.Background(), domain.DoorRiskDetectedEvent{
		RunID:       "run-123",
		StationCode: "PIN",
		Platform:    "P2",
		DoorNumber:  8,
		Timestamp:   time.Now(),
	})
	if err != nil {
		t.Fatalf("erro ao publicar evento: %v", err)
	}

	inspections, err := insRepo.List(context.Background(), 10, 0, "")
	if err != nil {
		t.Fatalf("erro ao listar inspecoes: %v", err)
	}

	if len(inspections) != 1 {
		t.Fatalf("esperava 1 inspecao gerada automaticamente, obteve %d", len(inspections))
	}

	ins := inspections[0]
	if ins.StationCode != "PIN" || ins.DoorNumber != 8 || ins.Priority != "critical" {
		t.Errorf("dados da inspecao incorretos: %+v", ins)
	}
}

func TestSubscribers_CacheInvalidationOnPredictionRun(t *testing.T) {
	bus := eventbus.NewInMemoryEventBus(nil)
	insRepo := newMockInspectionRepo()
	cache := &mockCacheInvalidator{}

	usecase.RegisterEventSubscribers(bus, insRepo, cache, nil)

	_ = bus.PublishSync(context.Background(), domain.PredictionRunCompletedEvent{
		RunID:       "run-10",
		HorizonDays: 7,
		TotalAtRisk: 5,
		Timestamp:   time.Now(),
	})

	if !cache.invalidated {
		t.Fatalf("esperava que o cache tivesse sido invalidado pelo subscriber")
	}
}
