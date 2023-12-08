package retry

import (
	"errors"
	"math"
	"math/rand"
	"testing"
	"time"
)

func TestDelayExponentialBackoff(t *testing.T) {
	testCases := []struct {
		attempt       int
		expectedDelay time.Duration
	}{
		{attempt: 1, expectedDelay: time.Nanosecond * 23_023_301},
		{attempt: 2, expectedDelay: time.Nanosecond * 46_046_602},
		{attempt: 3, expectedDelay: time.Nanosecond * 92_093_205},
		{attempt: 5, expectedDelay: time.Nanosecond * 368_372_823},
		{attempt: math.MaxInt64, expectedDelay: time.Second * 2},
	}

	for _, tc := range testCases {
		r := testRetry()
		delay := r.delay(tc.attempt)
		if delay != tc.expectedDelay {
			t.Errorf("expected delay %v, got %v for attempt %d", tc.expectedDelay, delay, tc.attempt)
		}
	}
}

func TestDelayMaxCapping(t *testing.T) {
	r := testRetry()
	highAttempt := r.MaxAttempts * 10
	if delay := r.delay(highAttempt); delay != r.MaxDelay {
		t.Errorf("expected delay to be capped at MaxDelay, got %v", delay)
	}
}

func TestDoMaxAttemptsError(t *testing.T) {
	r := testRetry()
	fn, _ := mockFunction(r.MaxAttempts + 1) // Ensure more failures than max attempts

	err := r.Do(fn)
	if !errors.Is(err, errMaxAttemptsReached) {
		t.Errorf("expected errMaxAttemptsReached, got %v", err)
	}
}

func TestDoAlwaysFail(t *testing.T) {
	r := testRetry()
	fn, attempts := mockFunction(1_000)

	start := time.Now()
	if err := r.Do(fn); !errors.Is(err, errMaxAttemptsReached) {
		t.Errorf("expected 'max attempts reached' error, got %v", err)
	}

	if elapsed := time.Since(start); elapsed > r.MaxDelay {
		t.Errorf("expected function to run less than MaxDelay but ran for %d", elapsed)
	}

	if *attempts != r.MaxAttempts {
		t.Errorf("expected number of attempts: got %v, want %v", attempts, r.MaxAttempts)
	}
}

func TestDoImmediateSuccess(t *testing.T) {
	r := testRetry()
	fn, attempts := mockFunction(0) // Succeeds on the first attempt

	if err := r.Do(fn); err != nil {
		t.Errorf("expected immediate success, got error %v", err)
	}

	if *attempts != 0 {
		t.Errorf("expected no attempts, got %v", *attempts)
	}
}

func TestDoTimeout(t *testing.T) {
	tests := []struct {
		name                  string
		failuresBeforeSuccess int
		timeout               time.Duration
		expectTimeout         bool
	}{
		{"no_timeout_err", 0, 1 * time.Second, false},
		{"timeout_err", 100, 50 * time.Millisecond, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := testRetry()
			fn, attempts := mockFunction(tt.failuresBeforeSuccess)
			err := r.DoTimeout(tt.timeout, fn)

			if tt.expectTimeout {
				if !errors.Is(err, errTimoutReached) {
					t.Errorf("expected timeout but got: %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if *attempts != tt.failuresBeforeSuccess {
					t.Errorf("unexpected number of attempts: got %v, want %v", attempts, tt.failuresBeforeSuccess)
				}
			}
		})
	}
}

// Test utility functions

func mockFunction(failuresBeforeSuccess int) (func() error, *int) {
	attempts := 0
	return func() error {
		if attempts < failuresBeforeSuccess {
			attempts++
			return errors.New("error")
		}
		return nil
	}, &attempts
}

func testRetry() *Retry {
	return &Retry{
		DelayFactor:         10 * time.Millisecond,
		MaxDelay:            2 * time.Second,
		MaxAttempts:         2,
		RandomizationFactor: 0.25,
		Rand:                rand.New(rand.NewSource(1)),
	}
}
