package eventbus

import (
	"context"
	"log/slog"
	"sync"

	"github.com/g04/backend/internal/domain"
)

var _ domain.EventBus = (*InMemoryEventBus)(nil)

type InMemoryEventBus struct {
	mu       sync.RWMutex
	handlers map[string][]domain.EventHandler
	logger   *slog.Logger
}

func NewInMemoryEventBus(logger *slog.Logger) *InMemoryEventBus {
	return &InMemoryEventBus{
		handlers: make(map[string][]domain.EventHandler),
		logger:   logger,
	}
}

func (b *InMemoryEventBus) Subscribe(eventName string, handler domain.EventHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventName] = append(b.handlers[eventName], handler)
}

func (b *InMemoryEventBus) Publish(ctx context.Context, event domain.Event) {
	b.mu.RLock()
	handlersList, exists := b.handlers[event.EventName()]
	b.mu.RUnlock()

	if !exists || len(handlersList) == 0 {
		return
	}

	for _, h := range handlersList {
		handler := h
		go func() {
			defer func() {
				if r := recover(); r != nil && b.logger != nil {
					b.logger.Error("panic capturado em subscriber de evento",
						slog.String("event", event.EventName()),
						slog.Any("recover", r),
					)
				}
			}()

			bgCtx := context.Background()
			if err := handler(bgCtx, event); err != nil && b.logger != nil {
				b.logger.Warn("erro ao processar evento de dominio",
					slog.String("event", event.EventName()),
					slog.String("error", err.Error()),
				)
			}
		}()
	}
}

func (b *InMemoryEventBus) PublishSync(ctx context.Context, event domain.Event) error {
	b.mu.RLock()
	handlersList, exists := b.handlers[event.EventName()]
	b.mu.RUnlock()

	if !exists || len(handlersList) == 0 {
		return nil
	}

	for _, h := range handlersList {
		if err := h(ctx, event); err != nil {
			return err
		}
	}
	return nil
}
