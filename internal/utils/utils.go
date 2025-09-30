package utils

import (
	"errors"
	"math"
	"math/rand/v2"
	"time"
)

func Must[T any](value T, err error) T {
	if err != nil {
		panic(err)
	}
	return value
}

func Backoff(fn func() error, maxRetries int) error {
	var errs []error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if err := fn(); err != nil {
			errs = append(errs, err)
			if attempt < maxRetries {
				maxBackoff := 30 * time.Second
				baseBackoff := min(time.Duration(math.Pow(2, float64(attempt)))*time.Second, maxBackoff)
				jitter := time.Duration(float64(baseBackoff) * (0.5 + 0.5*rand.Float64()))
				time.Sleep(jitter)
			}
		} else {
			return nil
		}
	}
	return errors.Join(errs...)
}
