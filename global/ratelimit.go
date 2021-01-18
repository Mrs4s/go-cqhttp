package global

import (
	"context"

	"golang.org/x/time/rate"
)

var limiter *rate.Limiter
var limitEnable = false

//RateLimit 执行API调用速率限制
func RateLimit(ctx context.Context) {
	if limitEnable {
		_ = limiter.Wait(ctx)
	}
}

//InitLimiter 初始化速率限制器
func InitLimiter(frequency float64, bucketSize int) {
	limitEnable = true
	limiter = rate.NewLimiter(rate.Limit(frequency), bucketSize)
}
