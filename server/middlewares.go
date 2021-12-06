package server

import (
	"container/list"
	"context"
	"sync"
	"time"

	"github.com/Mrs4s/go-cqhttp/coolq"
	"github.com/Mrs4s/go-cqhttp/global"
	"github.com/Mrs4s/go-cqhttp/modules/api"

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
	return func(_ string, _ api.Getter) global.MSG {
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
		queue.PushBack(event.RawMsg)
		for maxSize != 0 && queue.Len() > maxSize {
			queue.Remove(queue.Front())
		}
		cond.Signal()
	})
	return func(action string, p api.Getter) global.MSG {
		if action != "get_updates" {
			return nil
		}
		var (
			once    sync.Once
			ch      = make(chan []interface{}, 1)
			timeout = time.Duration(p.Get("timeout").Int()) * time.Second
		)
		defer close(ch)
		go func() {
			mutex.Lock()
			defer mutex.Unlock()
			if queue.Len() == 0 {
				cond.Wait()
			}
			once.Do(func() {
				limit := int(p.Get("limit").Int())
				if limit <= 0 || queue.Len() < limit {
					limit = queue.Len()
				}
				ret := make([]interface{}, limit)
				for i := 0; i < limit; i++ {
					ret[i] = queue.Remove(queue.Front())
				}
				ch <- ret
			})
		}()
		if timeout != 0 {
			select {
			case <-time.After(timeout):
				once.Do(func() {})
				return coolq.OK([]interface{}{})
			case ret := <-ch:
				return coolq.OK(ret)
			}
		}
		return coolq.OK(<-ch)
	}
}
