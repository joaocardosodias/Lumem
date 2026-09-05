package scheduler

import (
	"log/slog"
	"sync"

	"github.com/g04/backend/internal/usecase/port"
)

var _ port.JobRunner = (*GoroutineJobRunner)(nil)

type GoroutineJobRunner struct {
	wg     *sync.WaitGroup
	logger *slog.Logger
}

func NewGoroutineJobRunner(wg *sync.WaitGroup, logger ...*slog.Logger) *GoroutineJobRunner {
	var l *slog.Logger
	if len(logger) > 0 {
		l = logger[0]
	}
	return &GoroutineJobRunner{
		wg:     wg,
		logger: l,
	}
}

func (r *GoroutineJobRunner) Run(job func()) {
	if r.wg != nil {
		r.wg.Add(1)
	}
	go func() {
		if r.wg != nil {
			defer r.wg.Done()
		}
		defer func() {
			if rec := recover(); rec != nil {
				if r.logger != nil {
					r.logger.Error("panico recuperado no job runner", slog.Any("panic", rec))
				}
			}
		}()
		job()
	}()
}
