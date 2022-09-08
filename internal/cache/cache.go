// Package cache impl the cache for gocq
package cache

import (
	log "github.com/sirupsen/logrus"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

// Media Cache DBs
var (
	Image Cache
	Video Cache
	// todo: Voice?
)

// Cache wraps the btree.DB for concurrent safe
type Cache struct {
	ldb *leveldb.DB
}

// Insert 添加媒体缓存
func (c *Cache) Insert(md5, data []byte) {
	_ = c.ldb.Put(md5, data, nil)
}

// Get 获取缓存信息
func (c *Cache) Get(md5 []byte) []byte {
	got, _ := c.ldb.Get(md5, nil)
	return got
}

// Delete 删除指定缓存
func (c *Cache) Delete(md5 []byte) {
	_ = c.ldb.Delete(md5, nil)
}

// Init 初始化 Cache
func Init() {
	open := func(typ, path string, cache *Cache) {
		ldb, err := leveldb.OpenFile(path, &opt.Options{
			WriteBuffer: 4 * opt.KiB,
		})
		if err != nil {
			log.Fatalf("open cache %s db failed: %v", typ, err)
		}
		cache.ldb = ldb
	}
	open("image", "data/images", &Image)
	open("video", "data/videos", &Video)
}
