package server

import (
	"container/list"
	"context"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Mrs4s/go-cqhttp/coolq"
	"github.com/Mrs4s/go-cqhttp/global"

	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"golang.org/x/time/rate"
)

var (
	filters     = make(map[string]global.Filter)
	filterMutex sync.RWMutex
)

func rateLimit(frequency float64, bucketSize int) handler {
	limiter := rate.NewLimiter(rate.Limit(frequency), bucketSize)
	return func(_ string, _ resultGetter) global.MSG {
		_ = limiter.Wait(context.Background())
		return nil
	}
}

func addFilter(file string) {
	if file == "" {
		return
	}
	bs, err := os.ReadFile(file)
	if err != nil {
		log.Error("init filter error: ", err)
		return
	}
	defer func() {
		if err := recover(); err != nil {
			log.Error("init filter error: ", err)
		}
	}()
	filter := global.Generate("and", gjson.ParseBytes(bs))
	filterMutex.Lock()
	filters[file] = filter
	filterMutex.Unlock()
}

func findFilter(file string) global.Filter {
	if file == "" {
		return nil
	}
	filterMutex.RLock()
	defer filterMutex.RUnlock()
	return filters[file]
}

func longPolling(bot *coolq.CQBot, maxSize int) handler {
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
	return func(action string, p resultGetter) global.MSG {
		if action != "get_updates" {
			return nil
		}
		var (
			ok      int32
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
			if atomic.CompareAndSwapInt32(&ok, 0, 1) {
				limit := int(p.Get("limit").Int())
				if limit <= 0 || queue.Len() < limit {
					limit = queue.Len()
				}
				ret := make([]interface{}, limit)
				for i := 0; i < limit; i++ {
					ret[i] = queue.Remove(queue.Front())
				}
				ch <- ret
			}
		}()
		if timeout != 0 {
			select {
			case <-time.After(timeout):
				atomic.StoreInt32(&ok, 1)
				return coolq.OK([]interface{}{})
			case ret := <-ch:
				return coolq.OK(ret)
			}
		}
		return coolq.OK(<-ch)
	}
}
