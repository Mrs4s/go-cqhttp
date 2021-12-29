// Package cache impl the cache for gocq
package cache

import (
	"fmt"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/Mrs4s/go-cqhttp/global"
	"github.com/Mrs4s/go-cqhttp/internal/base"
	"github.com/Mrs4s/go-cqhttp/internal/btree"
)

// Media Cache DBs
var (
	Image Cache
	Video Cache
	// todo: Voice?
)

// Cache wraps the btree.DB for concurrent safe
type Cache struct {
	lock sync.RWMutex
	db   *btree.DB
}

// Insert 添加媒体缓存
func (c *Cache) Insert(md5, data []byte) {
	c.lock.Lock()
	defer c.lock.Unlock()

	var hash [16]byte
	copy(hash[:], md5)
	c.db.Insert(&hash[0], data)
}

// Get 获取缓存信息
func (c *Cache) Get(md5 []byte) []byte {
	c.lock.RLock()
	defer c.lock.RUnlock()

	var hash [16]byte
	copy(hash[:], md5)
	return c.db.Get(&hash[0])
}

// Delete 删除指定缓存
func (c *Cache) Delete(md5 []byte) {
	c.lock.Lock()
	defer c.lock.Unlock()

	var hash [16]byte
	copy(hash[:], md5)
	_ = c.db.Delete(&hash[0])
}

// Init 初始化 Cache
func Init() {
	node, ok := base.Database["cache"]
	var conf map[string]string
	if ok {
		err := node.Decode(&conf)
		if err != nil {
			log.Fatalf("failed to read cache config: %v", err)
		}
	}

	open := func(typ string, cache *Cache) {
		file := conf[typ]
		if file == "" {
			file = fmt.Sprintf("data/%s.db", typ)
		}
		if global.PathExists(file) {
			db, err := btree.Open(file)
			if err != nil {
				log.Fatalf("open %s cache failed: %v", typ, err)
			}
			cache.db = db
		} else {
			db, err := btree.Create(file)
			if err != nil {
				log.Fatalf("create %s cache failed: %v", typ, err)
			}
			cache.db = db
		}
	}
	open("image", &Image)
	open("video", &Video)
}
