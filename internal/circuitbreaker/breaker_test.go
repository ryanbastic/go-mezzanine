package circuitbreaker

import (
	"errors"
	"sync"
	"testing"
	"time"
)

var errTest = errors.New("test error")

func TestNew(t *testing.T) {
	b := New(5, 30*time.Second)
	if b == nil {
		t.Fatal("New returned nil")
	}
	if b.GetState() != Closed {
		t.Errorf("initial state: got %d, want Closed(%d)", b.GetState(), Closed)
	}
}

func TestExecute_Success(t *testing.T) {
	b := New(3, time.Second)

	err := b.Execute(func() error {
		return nil
	})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if b.GetState() != Closed {
		t.Errorf("state should be Closed after success")
	}
}

func TestExecute_PropagatesError(t *testing.T) {
	b := New(3, time.Second)

	err := b.Execute(func() error {
		return errTest
	})
	if !errors.Is(err, errTest) {
		t.Errorf("expected errTest, got %v", err)
	}
}

func TestExecute_OpensAfterMaxFailures(t *testing.T) {
	b := New(3, time.Second)

	// Trigger 3 failures to open the breaker
	for i := 0; i < 3; i++ {
		b.Execute(func() error { return errTest })
	}

	if b.GetState() != Open {
		t.Fatalf("state should be Open after %d failures, got %d", 3, b.GetState())
	}

	// Next call should be rejected immediately
	err := b.Execute(func() error {
		t.Error("function should not be called when circuit is open")
		return nil
	})
	if !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestExecute_DoesNotOpenBelowMaxFailures(t *testing.T) {
	b := New(5, time.Second)

	for i := 0; i < 4; i++ {
		b.Execute(func() error { return errTest })
	}

	if b.GetState() != Closed {
		t.Errorf("state should still be Closed after 4/%d failures", 5)
	}
}

func TestExecute_SuccessResetsFailureCount(t *testing.T) {
	b := New(3, time.Second)

	// 2 failures
	for i := 0; i < 2; i++ {
		b.Execute(func() error { return errTest })
	}

	// 1 success resets
	b.Execute(func() error { return nil })

	// 2 more failures - should still be closed (total 2, not 4)
	for i := 0; i < 2; i++ {
		b.Execute(func() error { return errTest })
	}

	if b.GetState() != Closed {
		t.Error("state should be Closed after success reset")
	}
}

func TestExecute_HalfOpenTransition(t *testing.T) {
	b := New(2, 10*time.Millisecond)

	// Open the breaker
	for i := 0; i < 2; i++ {
		b.Execute(func() error { return errTest })
	}
	if b.GetState() != Open {
		t.Fatal("expected Open state")
	}

	// Wait for reset timeout
	time.Sleep(20 * time.Millisecond)

	// Next call should transition to HalfOpen and execute
	called := false
	err := b.Execute(func() error {
		called = true
		return nil
	})
	if err != nil {
		t.Errorf("expected no error in half-open, got %v", err)
	}
	if !called {
		t.Error("function should have been called in half-open state")
	}
	if b.GetState() != Closed {
		t.Errorf("state should be Closed after half-open success, got %d", b.GetState())
	}
}

func TestExecute_HalfOpenFailure_ReOpens(t *testing.T) {
	b := New(2, 10*time.Millisecond)

	// Open the breaker
	for i := 0; i < 2; i++ {
		b.Execute(func() error { return errTest })
	}

	// Wait for reset timeout
	time.Sleep(20 * time.Millisecond)

	// Fail in half-open — should re-open
	b.Execute(func() error { return errTest })

	if b.GetState() != Open {
		t.Errorf("state should be Open after half-open failure, got %d", b.GetState())
	}
}

func TestExecute_OpenRejectsBeforeTimeout(t *testing.T) {
	b := New(1, time.Hour) // Very long timeout

	b.Execute(func() error { return errTest })

	if b.GetState() != Open {
		t.Fatal("expected Open")
	}

	err := b.Execute(func() error {
		t.Error("should not be called")
		return nil
	})
	if !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestGetState_ThreadSafe(t *testing.T) {
	b := New(100, time.Second)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b.GetState()
		}()
	}
	wg.Wait()
}

func TestExecute_ConcurrentAccess(t *testing.T) {
	b := New(100, time.Second)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			if n%2 == 0 {
				b.Execute(func() error { return nil })
			} else {
				b.Execute(func() error { return errTest })
			}
		}(i)
	}
	wg.Wait()
}

func TestState_Constants(t *testing.T) {
	if Closed != 0 {
		t.Errorf("Closed should be 0, got %d", Closed)
	}
	if Open != 1 {
		t.Errorf("Open should be 1, got %d", Open)
	}
	if HalfOpen != 2 {
		t.Errorf("HalfOpen should be 2, got %d", HalfOpen)
	}
}

func TestExecute_SingleFailureDoesNotOpen(t *testing.T) {
	b := New(5, time.Second)

	b.Execute(func() error { return errTest })

	if b.GetState() != Closed {
		t.Error("single failure should not open breaker with maxFailures=5")
	}
}

func TestExecute_ExactMaxFailuresOpens(t *testing.T) {
	for maxF := 1; maxF <= 5; maxF++ {
		b := New(maxF, time.Second)
		for i := 0; i < maxF; i++ {
			b.Execute(func() error { return errTest })
		}
		if b.GetState() != Open {
			t.Errorf("maxFailures=%d: expected Open after exactly %d failures", maxF, maxF)
		}
	}
}

func TestErrCircuitOpen_IsError(t *testing.T) {
	var err error = ErrCircuitOpen
	if err.Error() != "circuit breaker is open" {
		t.Errorf("unexpected error message: %q", err.Error())
	}
}

func TestExecute_SuccessAfterReopen(t *testing.T) {
	b := New(1, 10*time.Millisecond)

	// Open
	b.Execute(func() error { return errTest })
	if b.GetState() != Open {
		t.Fatal("expected Open")
	}

	// Wait and recover
	time.Sleep(20 * time.Millisecond)

	err := b.Execute(func() error { return nil })
	if err != nil {
		t.Errorf("expected success after recovery, got %v", err)
	}
	if b.GetState() != Closed {
		t.Error("expected Closed after successful recovery")
	}

	// Open again
	b.Execute(func() error { return errTest })
	if b.GetState() != Open {
		t.Fatal("expected Open again")
	}

	// Recover again
	time.Sleep(20 * time.Millisecond)
	err = b.Execute(func() error { return nil })
	if err != nil {
		t.Errorf("expected success on second recovery, got %v", err)
	}
	if b.GetState() != Closed {
		t.Error("expected Closed after second recovery")
	}
}
