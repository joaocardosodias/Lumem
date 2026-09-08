package usecase

import (
	"context"
	"errors"
	"testing"
)

func TestConsumeBatchesBoundedAndStopsOnDatabaseFailure(t *testing.T) {
	sentinel := errors.New("database unavailable")
	seen, calls := 0, 0
	processed, inserted, err := consumeBatches(context.Background(), func(yield func(int) error) error {
		for i := 0; i < 1501; i++ {
			seen++
			if err := yield(i); err != nil {
				return err
			}
		}
		return nil
	}, func(batch []int) (int, error) {
		calls++
		if len(batch) > 500 {
			t.Fatal("unbounded batch")
		}
		if calls == 2 {
			return 0, sentinel
		}
		return len(batch), nil
	})
	var permanent *permanentIngestionError
	if !errors.Is(err, sentinel) || errors.As(err, &permanent) || seen != 1000 || processed != 1000 || inserted != 500 {
		t.Fatalf("database failure must stop streaming and remain retryable: seen=%d processed=%d inserted=%d err=%v", seen, processed, inserted, err)
	}
}

func TestConsumeBatchesFlushesTailAndClassifiesErrors(t *testing.T) {
	sizes := []int{}
	processed, inserted, err := consumeBatches(context.Background(), func(yield func(int) error) error {
		for i := 0; i < 1001; i++ {
			if err := yield(i); err != nil {
				return err
			}
		}
		return nil
	}, func(batch []int) (int, error) { sizes = append(sizes, len(batch)); return len(batch), nil })
	if err != nil || processed != 1001 || inserted != 1001 || len(sizes) != 3 || sizes[2] != 1 {
		t.Fatalf("incorrect tail: %v %v", sizes, err)
	}
	_, _, err = consumeBatches(context.Background(), func(func(int) error) error { return errors.New("malformed CSV") }, func([]int) (int, error) { t.Fatal("insert after parsing failure"); return 0, nil })
	var permanent *permanentIngestionError
	if !errors.As(err, &permanent) {
		t.Fatalf("expected permanent error: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err = consumeBatches(ctx, func(yield func(int) error) error { return yield(1) }, func([]int) (int, error) { t.Fatal("insert after cancellation"); return 0, nil })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation: %v", err)
	}
}
