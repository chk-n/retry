package retry

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"
)

type randGenerator interface {
	Float64() float64
}

type Retry struct {
	// Base delay used in exponential backoff, should be greater than 0
	DelayFactor time.Duration
	// Should be between 0 and 1, setting this to 0 leads to no randomization
	RandomizationFactor float64
	// Maximum delay to wait between each retry call
	MaxDelay time.Duration
	// Maximum number of attempts before returning error
	MaxAttempts int
	// random number generator. We expect values to call Float64() to be within [0.0, 1.0]
	Rand randGenerator
}

// Creates default retry policy
func NewDefault() *Retry {
	return &Retry{
		DelayFactor:         100 * time.Millisecond,
		RandomizationFactor: 0.25,
		MaxDelay:            10 * time.Second,
		MaxAttempts:         8,
		Rand:                rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Computes delay after a given number of attempts.
func (r *Retry) delay(attempt int) time.Duration {
	if attempt > 49 {
		// set to floored log(max.Int64) to ensure math.Pow doesnt return +Inf
		attempt = 49
	}

	rf := r.RandomizationFactor*(r.Rand.Float64()) + 1
	var delay time.Duration

	b := math.Pow(2, float64(attempt))
	fmt.Println(b)
	if b > math.MaxFloat64 {
		delay = time.Duration(math.MaxInt64)
	} else {
		d := float64(r.DelayFactor) * b * rf
		fmt.Println(d)
		if d > math.MaxFloat64 {
			delay = time.Duration(math.MaxInt64)
		} else {
			delay = time.Duration(d)
		}
	}
	fmt.Println(delay.Milliseconds())
	if delay < r.MaxDelay {
		return delay
	}
	return r.MaxDelay
}

// Calls the function fn, retrying up to MaxAttempts times.
// Returns errMaxAttemptsReached wrapped with last error returned by fn()
func (r *Retry) Do(fn func() error) error {
	var err error
	for attempt := 1; attempt <= r.MaxAttempts; attempt++ {
		err = fn()
		if err == nil {
			return nil
		}

		time.Sleep(r.delay(attempt))
	}
	return errors.Join(errMaxAttemptsReached, err)
}

// Calls function fn, retrying up to MaxAttempts times or until timeout reached.
func (r *Retry) DoTimeout(timeout time.Duration, fn func() error) error {
	done := make(chan interface{})
	var err error
	go func() {
		err = r.Do(fn)
		done <- struct{}{}
	}()

	select {
	case <-time.After(timeout):
		return errTimoutReached
	case <-done:
		return err
	}

}

var (
	errMaxAttemptsReached = fmt.Errorf("retry: max attempts reached")
	errTimoutReached      = fmt.Errorf("retry: timeout reached")
)
