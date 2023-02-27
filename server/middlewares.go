package server

import (
	"container/list"
	"context"
	"sync"
	"time"

	"github.com/Mrs4s/go-cqhttp/coolq"
	"github.com/Mrs4s/go-cqhttp/global"
	"github.com/Mrs4s/go-cqhttp/modules/api"
	"github.com/Mrs4s/go-cqhttp/pkg/onebot"

	"golang.org/x/time/rate"
)

// MiddleWares 通信中间件
type MiddleWares struct {
	AccessToken string `yaml:"access-token"`
	Filter      string `yaml:"filter"`
	RateLimit   struct {
		Enabled   bool    `yaml:"enabled"`
		Frequency float64 `yaml:"frequency"`
		Bucket    int     `yaml:"bucket"`
	} `yaml:"rate-limit"`
}

func rateLimit(frequency float64, bucketSize int) api.Handler {
	limiter := rate.NewLimiter(rate.Limit(frequency), bucketSize)
	return func(_ string, _ *onebot.Spec, _ api.Getter) global.MSG {
		_ = limiter.Wait(context.Background())
		return nil
	}
}

func longPolling(bot *coolq.CQBot, maxSize int) api.Handler {
	var mutex sync.Mutex
	cond := sync.NewCond(&mutex)
	queue := list.New()
	bot.OnEventPush(func(event *coolq.Event) {
		mutex.Lock()
		defer mutex.Unlock()
		queue.PushBack(event.Raw)
		for maxSize != 0 && queue.Len() > maxSize {
			queue.Remove(queue.Front())
		}
		cond.Signal()
	})
	return func(action string, spec *onebot.Spec, p api.Getter) global.MSG {
		switch {
		case spec.Version == 11 && action == "get_updates": // ok
		case spec.Version == 12 && action == "get_latest_events": // ok
		default:
			return nil
		}
		var (
			ch      = make(chan []any)
			timeout = time.Duration(p.Get("timeout").Int()) * time.Second
		)
		go func() {
			mutex.Lock()
			defer mutex.Unlock()
			for queue.Len() == 0 {
				cond.Wait()
			}
			limit := int(p.Get("limit").Int())
			if limit <= 0 || queue.Len() < limit {
				limit = queue.Len()
			}
			ret := make([]any, limit)
			elem := queue.Front()
			for i := 0; i < limit; i++ {
				ret[i] = elem.Value
				elem = elem.Next()
			}
			select {
			case ch <- ret:
				for i := 0; i < limit; i++ { // remove sent msg
					queue.Remove(queue.Front())
				}
			default:
				// don't block if parent already return due to timeout
			}
		}()
		if timeout != 0 {
			select {
			case <-time.After(timeout):
				return coolq.OK([]any{})
			case ret := <-ch:
				return coolq.OK(ret)
			}
		}
		return coolq.OK(<-ch)
	}
}
