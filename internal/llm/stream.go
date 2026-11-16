package llm

import (
	"context"
	"errors"

	"github.com/CrisH2307/repetition-study/internal/domain"
)

// errNoRetry marks an error that must not be retried.
type errNoRetry struct{ error }

func (e errNoRetry) Unwrap() error { return e.error }

// streamRetry runs attempt in a goroutine and feeds its text to the returned
// channel. It retries only until the first text arrives. After that a failure
// is sent as a terminal Delta.Err, because half a question cannot be replayed.
func streamRetry(ctx context.Context, r Retry, attempt func(ctx context.Context, emit func(string) error) error) <-chan domain.Delta {
	out := make(chan domain.Delta, 16)
	go func() {
		defer close(out)
		started := false
		err := r.do(ctx, func(actx context.Context) error {
			err := attempt(actx, func(t string) error {
				started = true
				select {
				case out <- domain.Delta{Text: t}:
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			})
			if err != nil && started {
				return errNoRetry{err}
			}
			return err
		})
		if err != nil {
			var nr errNoRetry
			if errors.As(err, &nr) {
				err = nr.error
			}
			out <- domain.Delta{Err: err}
		}
	}()
	return out
}
