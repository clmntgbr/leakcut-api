package messaging

import "context"

type attemptContextKey struct{}

type Attempt struct {
	Count int
	Max   int
}

func WithAttempt(ctx context.Context, count, max int) context.Context {
	return context.WithValue(ctx, attemptContextKey{}, Attempt{Count: count, Max: max})
}

func LastAttempt(ctx context.Context) bool {
	value, ok := ctx.Value(attemptContextKey{}).(Attempt)
	if !ok {
		return false
	}
	return value.Count >= value.Max
}
