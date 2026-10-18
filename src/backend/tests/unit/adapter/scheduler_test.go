package adapter_test

import (
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/g04/backend/internal/adapter/scheduler"
)

func TestGoroutineJobRunner_NormalRun(t *testing.T) {
	var wg sync.WaitGroup
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	runner := scheduler.NewGoroutineJobRunner(&wg, logger)

	var executed int32
	runner.Run(func() {
		atomic.AddInt32(&executed, 1)
	})

	wg.Wait()
	if atomic.LoadInt32(&executed) != 1 {
		t.Fatalf("expected job to execute once, got %d", executed)
	}
}

func TestGoroutineJobRunner_NilWaitGroupAndLogger(t *testing.T) {
	runner := scheduler.NewGoroutineJobRunner(nil)

	var executed int32
	runner.Run(func() {
		atomic.AddInt32(&executed, 1)
	})

	time.Sleep(50 * time.Millisecond)
	if atomic.LoadInt32(&executed) != 1 {
		t.Fatalf("expected job to execute once, got %d", executed)
	}
}

func TestGoroutineJobRunner_PanicRecovery(t *testing.T) {
	var wg sync.WaitGroup
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	runner := scheduler.NewGoroutineJobRunner(&wg, logger)

	runner.Run(func() {
		panic("forced panic test")
	})

	wg.Wait()
}
