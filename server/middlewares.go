package server

import (
	"context"
	"os"
	"sync"

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
	return func(_ string, _ resultGetter) coolq.MSG {
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
