package usecase_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
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

func TestSubscribers_DoorRiskAlertReceived(t *testing.T) {
	bus := eventbus.NewInMemoryEventBus(nil)
	cache := &mockCacheInvalidator{}

	usecase.RegisterEventSubscribers(bus, cache, nil)

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
}

func TestSubscribers_CacheInvalidationOnPredictionRun(t *testing.T) {
	bus := eventbus.NewInMemoryEventBus(nil)
	cache := &mockCacheInvalidator{}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	usecase.RegisterEventSubscribers(bus, cache, logger)

	_ = bus.PublishSync(context.Background(), domain.PredictionRunCompletedEvent{
		RunID:       "run-10",
		HorizonDays: 7,
		TotalAtRisk: 5,
		Timestamp:   time.Now(),
	})

	if !cache.invalidated {
		t.Fatalf("esperava que o cache tivesse sido invalidado pelo subscriber")
	}

	_ = bus.PublishSync(context.Background(), domain.DoorRiskDetectedEvent{
		RunID:       "run-11",
		StationCode: "FAL",
		Platform:    "P1",
		DoorNumber:  4,
		Timestamp:   time.Now(),
	})

	_ = bus.PublishSync(context.Background(), dummyTestEvent{name: "DoorRiskDetected"})
	_ = bus.PublishSync(context.Background(), dummyTestEvent{name: "PredictionRunCompleted"})
}

func TestSubscribers_FileUploadCompleted(t *testing.T) {
	bus := eventbus.NewInMemoryEventBus(nil)
	cache := &mockCacheInvalidator{}

	usecase.RegisterEventSubscribers(bus, cache, nil)

	_ = bus.PublishSync(context.Background(), domain.FileUploadCompletedEvent{
		UploadID:  "up-123",
		Timestamp: time.Now(),
	})

	if !cache.invalidated {
		t.Fatalf("expected cache invalidation on file upload completed")
	}
}

type dummyTestEvent struct {
	name string
}

func (d dummyTestEvent) EventName() string     { return d.name }
func (d dummyTestEvent) OccurredAt() time.Time { return time.Now() }

func TestInMemoryEventBus_EdgeBranches(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	bus := eventbus.NewInMemoryEventBus(logger)

	bus.Publish(context.Background(), dummyTestEvent{name: "NoHandlers"})

	if err := bus.PublishSync(context.Background(), dummyTestEvent{name: "NoHandlers"}); err != nil {
		t.Fatalf("esperava nil ao publicar sync sem handlers: %v", err)
	}

	bus.Subscribe("PanicTopic", func(ctx context.Context, e domain.Event) error {
		panic("erro simulado em subscriber")
	})

	bus.Subscribe("ErrorTopic", func(ctx context.Context, e domain.Event) error {
		return errors.New("falha no handler async")
	})

	bus.Subscribe("SyncErrorTopic", func(ctx context.Context, e domain.Event) error {
		return errors.New("falha sync")
	})

	bus.Publish(context.Background(), dummyTestEvent{name: "PanicTopic"})
	bus.Publish(context.Background(), dummyTestEvent{name: "ErrorTopic"})

	if err := bus.PublishSync(context.Background(), dummyTestEvent{name: "SyncErrorTopic"}); err == nil {
		t.Fatalf("esperava erro no PublishSync")
	}

	time.Sleep(50 * time.Millisecond)
}
