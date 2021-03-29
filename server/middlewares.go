package server

import (
	"context"

	"golang.org/x/time/rate"

	"github.com/Mrs4s/go-cqhttp/coolq"
)

func rateLimit(frequency float64, bucketSize int) handler {
	limiter := rate.NewLimiter(rate.Limit(frequency), bucketSize)
	return func(_ string, _ resultGetter) coolq.MSG {
		_ = limiter.Wait(context.Background())
		return nil
	}
}
