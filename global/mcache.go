package global

import (
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"path"
	"sync"
)

// mcache.go 用于缓存文件的简单记录与操作

// FileMapCache 记录当前正在写入的文件名称 让其存在于磁盘中而不执行删除
// 防止删除正在上传的文件而导致操作失败
type FileMapCache struct {
	CacheMap sync.Map      //防止并发影响"写"数据
	Lock     *sync.RWMutex //防止时序影响"读"数据
}

// CacheFileStat 缓存统计
type CacheFileStat struct {
	Count int32 //缓存文件数量
	Size  int64 //缓存大小 KB
}

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
		dirStat.Size += info.Size()
	}

	return dirStat, nil
}

// CacheStat 全量统计
func (c *FileMapCache) CacheStat() (*CacheFileStat, error) {
	var stat = new(CacheFileStat)

	var dirArr = []string{CachePath, ImagePath, VoicePath, VideoPath}
	for _, dir := range dirArr {
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
func (c *FileMapCache) Clean() {
	c.Lock.Lock()
	defer c.Lock.Unlock()

	{
		// 日志在控制台输出 而不是在接口中输出
		go c.removeDirFile(CachePath)
		go c.removeDirFile(ImagePath)
		go c.removeDirFile(VoicePath)
		go c.removeDirFile(VideoPath)
	}
}

func (c *FileMapCache) removeDirFile(dir string) {
	cacheDir, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Errorf("read cache dir err: %v", err)
		return
	}

	for _, info := range cacheDir {
		fullname := path.Join(dir, info.Name())
		if _, exist := c.CacheMap.Load(fullname); exist {
			// 存在 跳过
			continue
		}
		// 删除
		DelFile(fullname)
	}
}

func (c *FileMapCache) Store(key string) {
	c.Lock.Lock()
	defer c.Lock.Unlock()

	c.CacheMap.Store(key, true)
}

func (c *FileMapCache) Delete(key string) {
	c.Lock.Lock()
	defer c.Lock.Unlock()

	c.CacheMap.Delete(key)
}
