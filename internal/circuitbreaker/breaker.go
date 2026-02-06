package circuitbreaker

import (
	"errors"
	"sync"
	"time"
)

// State represents the circuit breaker state.
type State int

const (
	Closed   State = iota // Normal operation — requests pass through.
	Open                  // Failing — requests are rejected immediately.
	HalfOpen              // Testing recovery — one request allowed through.
)

// ErrCircuitOpen is returned when the circuit breaker is open.
var ErrCircuitOpen = errors.New("circuit breaker is open")

// Breaker implements the circuit breaker pattern.
type Breaker struct {
	mu              sync.Mutex
	state           State
	failures        int
	maxFailures     int
	resetTimeout    time.Duration
	lastFailureTime time.Time
}

// New creates a Breaker that opens after maxFailures consecutive errors
// and attempts recovery after resetTimeout.
func New(maxFailures int, resetTimeout time.Duration) *Breaker {
	return &Breaker{
		state:        Closed,
		maxFailures:  maxFailures,
		resetTimeout: resetTimeout,
	}
}

// Execute runs fn through the circuit breaker. If the circuit is open,
// ErrCircuitOpen is returned without calling fn.
func (b *Breaker) Execute(fn func() error) error {
	b.mu.Lock()
	switch b.state {
	case Open:
		if time.Since(b.lastFailureTime) > b.resetTimeout {
			b.state = HalfOpen
		} else {
			b.mu.Unlock()
			return ErrCircuitOpen
		}
	}
	b.mu.Unlock()

	err := fn()

	b.mu.Lock()
	defer b.mu.Unlock()

	if err != nil {
		b.failures++
		b.lastFailureTime = time.Now()
		if b.failures >= b.maxFailures {
			b.state = Open
		}
		return err
	}

	b.failures = 0
	b.state = Closed
	return nil
}

// State returns the current state of the breaker.
func (b *Breaker) GetState() State {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.state
}
