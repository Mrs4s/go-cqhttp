package global

import (
	"errors"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"path"
	"sync"
)

const (
	// MCacheStatDefault API默认状态(未执行)
	MCacheStatDefault = iota
	// MCacheStatRunning API正在执行中
	MCacheStatRunning
)

// FileMapCache 记录当前正在写入的文件名称 让其存在于磁盘中而不执行删除
//
// 防止删除正在上传的文件而导致操作失败
type FileMapCache struct {
	CacheMap sync.Map      //防止并发影响"写"数据
	Lock     *sync.RWMutex //防止时序影响"读"数据
	Stat     uint8         //接口状态 防止多次重入
}

// CacheFileStat 缓存统计
type CacheFileStat struct {
	Count uint32 //缓存文件数量
	Size  uint64 //缓存大小 KB
}

// NewCacheFileMap 新的缓存文件清理对象
func NewCacheFileMap() *FileMapCache {
	return &FileMapCache{
		CacheMap: sync.Map{},
		Lock:     &sync.RWMutex{},
	}
}

// stat 指定目录 进行统计
func (c *FileMapCache) stat(dir string) (dirStat *CacheFileStat, err error) {
	dirStat = new(CacheFileStat)
	cacheDir, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Errorf("read cache dir err: %v", err)
		return nil, err
	}

	for _, info := range cacheDir {
		dirStat.Count++
		dirStat.Size += uint64(info.Size())
	}

	return dirStat, nil
}

// CacheStat 全量统计
func (c *FileMapCache) CacheStat() (*CacheFileStat, error) {
	var stat = new(CacheFileStat)

	var statDir = []string{CachePath, ImagePath, VoicePath, VideoPath}

	for _, dir := range statDir {
		temp, err := c.stat(dir)
		if err != nil {
			// 统计失败了 跳过该文件夹
			continue
		}
		stat.Size += temp.Size
		stat.Count += temp.Count
	}

	stat.Size /= 1024
	return stat, nil
}

// Clean 执行目录缓存清空
func (c *FileMapCache) Clean() error {
	c.Lock.Lock()
	defer c.Lock.Unlock()

	// 避免不必要协程开销
	if c.Stat == MCacheStatRunning {
		return errors.New("please do not try again, proces:clean_cache is running")
	}

	c.Stat = MCacheStatRunning
	var cleanDir = []string{CachePath, ImagePath, VoicePath, VideoPath}

	// 日志在控制台输出 而不是在接口中输出
	var wg = new(sync.WaitGroup)
	for _, dir := range cleanDir {
		wg.Add(1)
		go func(wg *sync.WaitGroup, dir string) {
			cacheDir, err := ioutil.ReadDir(dir)
			if err != nil {
				log.Errorf("read cache dir err: %v", err)
				return
			}

			for _, info := range cacheDir {
				fullname := path.Join(dir, info.Name())
				if _, exist := c.CacheMap.Load(fullname); exist {
					// 存在 跳过
					wg.Done()
					continue
				}
				// 删除
				DelFile(fullname)
			}
			wg.Done()
		}(wg, dir)
	}

	wg.Wait()
	c.Stat = MCacheStatDefault

	return nil
}

// Store 文件路径入map
func (c *FileMapCache) Store(key string) {
	c.Lock.Lock()
	defer c.Lock.Unlock()

	c.CacheMap.Store(key, true)
}

// Delete 文件路径删除
func (c *FileMapCache) Delete(key string) {
	c.Lock.Lock()
	defer c.Lock.Unlock()

	c.CacheMap.Delete(key)
}
