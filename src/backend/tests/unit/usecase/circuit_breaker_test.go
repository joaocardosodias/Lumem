package usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/g04/backend/internal/adapter/mlclient"
)

func TestCircuitBreaker_StateTransitions(t *testing.T) {
	cb := mlclient.NewCircuitBreaker(2, 50*time.Millisecond)

	err := cb.Execute(func() error {
		return nil
	})
	if err != nil {
		t.Fatalf("esperava sucesso na execucao inicial: %v", err)
	}
	if cb.State() != mlclient.StateClosed {
		t.Errorf("esperava StateClosed, obteve %v", cb.State())
	}

	_ = cb.Execute(func() error {
		return errors.New("falha 1")
	})
	_ = cb.Execute(func() error {
		return errors.New("falha 2")
	})

	if cb.State() != mlclient.StateOpen {
		t.Fatalf("esperava StateOpen apos 2 falhas, obteve %v", cb.State())
	}

	err = cb.Execute(func() error {
		return nil
	})
	if !errors.Is(err, mlclient.ErrCircuitOpen) {
		t.Fatalf("esperava ErrCircuitOpen enquanto circuito aberto, obteve: %v", err)
	}

	time.Sleep(60 * time.Millisecond)

	err = cb.Execute(func() error {
		return nil
	})
	if err != nil {
		t.Fatalf("esperava sucesso na recuperacao: %v", err)
	}

	if cb.State() != mlclient.StateClosed {
		t.Fatalf("esperava StateClosed apos sucesso em half-open, obteve %v", cb.State())
	}

	_ = cb.Execute(func() error { return errors.New("falha 1") })
	_ = cb.Execute(func() error { return errors.New("falha 2") })
	if cb.State() != mlclient.StateOpen {
		t.Fatalf("esperava StateOpen")
	}
	time.Sleep(60 * time.Millisecond)

	_ = cb.Execute(func() error { return errors.New("falha em half-open") })
	if cb.State() != mlclient.StateOpen {
		t.Fatalf("esperava retorno imediato a StateOpen apos falha em half-open, obteve %v", cb.State())
	}
}
