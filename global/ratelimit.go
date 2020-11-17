package global

import (
	"context"

	"golang.org/x/time/rate"
)

var limiter *rate.Limiter
var limitEnable = false

func RateLimit(ctx context.Context) {
	if limitEnable {
		_ = limiter.Wait(ctx)
	}
}

func InitLimiter(r float64, b int) {
	limitEnable = true
	limiter = rate.NewLimiter(rate.Limit(r), b)
}
