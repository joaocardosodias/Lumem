package mlclient

import (
	"errors"
	"sync"
	"time"
)

var ErrCircuitOpen = errors.New("circuito de ml aberto: vps temporariamente indisponivel")

type State int

const (
	StateClosed State = iota
	StateHalfOpen
	StateOpen
)

type CircuitBreaker struct {
	mu              sync.Mutex
	state           State
	failures        int
	threshold       int
	timeout         time.Duration
	lastStateChange time.Time
}

func NewCircuitBreaker(threshold int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:           StateClosed,
		threshold:       threshold,
		timeout:         timeout,
		lastStateChange: time.Now(),
	}
}

func (cb *CircuitBreaker) Execute(operation func() error) error {
	cb.mu.Lock()
	now := time.Now()

	if cb.state == StateOpen {
		if now.Sub(cb.lastStateChange) > cb.timeout {
			cb.state = StateHalfOpen
			cb.lastStateChange = now
		} else {
			cb.mu.Unlock()
			return ErrCircuitOpen
		}
	}
	cb.mu.Unlock()

	var err error
	maxRetries := 2
	baseDelay := 50 * time.Millisecond

	for attempt := 0; attempt <= maxRetries; attempt++ {
		err = operation()
		if err == nil {
			break
		}
		if attempt < maxRetries {
			time.Sleep(baseDelay * time.Duration(1<<attempt))
		}
	}

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.failures++
		if cb.failures >= cb.threshold || cb.state == StateHalfOpen {
			cb.state = StateOpen
			cb.lastStateChange = time.Now()
		}
		return err
	}

	cb.failures = 0
	cb.state = StateClosed
	return nil
}

func (cb *CircuitBreaker) State() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}
