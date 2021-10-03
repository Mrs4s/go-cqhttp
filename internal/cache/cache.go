// Package cache impl the cache for gocq
package cache

import (
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/Mrs4s/go-cqhttp/global"
	"github.com/Mrs4s/go-cqhttp/internal/base"
	"github.com/Mrs4s/go-cqhttp/internal/btree"
)

// todo(wdvxdr): always enable db-cache in v1.0.0

// EnableCacheDB 是否启用 btree db缓存图片等
var EnableCacheDB bool

// Media Cache DBs
var (
	Image *Cache
	Video *Cache
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

// Init 初始化 Cache
func Init() {
	node, ok := base.Database["cache"]
	if !ok {
		return
	}
	EnableCacheDB = true
	var conf map[string]string
	err := node.Decode(&conf)
	if err != nil {
		log.Fatalf("failed to read cache config: %v", err)
	}
	if conf == nil {
		conf = make(map[string]string)
	}
	if conf["image"] == "" {
		conf["image"] = "data/image.db"
	}
	if conf["video"] == "" {
		conf["video"] = "data/video.db"
	}

	var open = func(typ string, cache **Cache) {
		if global.PathExists(conf[typ]) {
			db, err := btree.Open(conf[typ])
			if err != nil {
				log.Fatalf("open %s cache failed: %v", typ, err)
			}
			*cache = &Cache{db: db}
		} else {
			db, err := btree.Create(conf[typ])
			if err != nil {
				log.Fatalf("create %s cache failed: %v", typ, err)
			}
			*cache = &Cache{db: db}
		}
	}
	open("image", &Image)
	open("video", &Video)
}
