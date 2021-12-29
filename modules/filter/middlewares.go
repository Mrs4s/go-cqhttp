package filter

import (
	"os"
	"sync"

	"github.com/tidwall/gjson"

	"github.com/Mrs4s/go-cqhttp/internal/log"
)

var (
	filters     = make(map[string]Filter)
	filterMutex sync.RWMutex
)

// Add adds a filter to the list of filters
func Add(file string) {
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
	filter := Generate("and", gjson.ParseBytes(bs))
	filterMutex.Lock()
	filters[file] = filter
	filterMutex.Unlock()
}

// Find returns the filter for the given file
func Find(file string) Filter {
	if file == "" {
		return nil
	}
	filterMutex.RLock()
	defer filterMutex.RUnlock()
	return filters[file]
}
