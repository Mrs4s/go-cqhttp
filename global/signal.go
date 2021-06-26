package global

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

var (
	mainStopCh chan struct{}
	mainOnce   sync.Once

	dumpMutex sync.Mutex
)

func dumpStack() {
	dumpMutex.Lock()
	defer dumpMutex.Unlock()

	log.Info("开始 dump 当前 goroutine stack 信息")

	buf := make([]byte, 1024)
	for {
		n := runtime.Stack(buf, true)
		if n < len(buf) {
			buf = buf[:n]
			break
		}
		buf = make([]byte, 2*len(buf))
	}

	fileName := fmt.Sprintf("%s.%d.stacks.%d.log", filepath.Base(os.Args[0]), os.Getpid(), time.Now().Unix())
	fd, err := os.Create(fileName)
	if err != nil {
		log.Errorf("保存 stackdump 到文件时出现错误: %v", err)
		log.Warnf("无法保存 stackdump. 将直接打印\n %s", buf)
		return
	}
	defer fd.Close()
	_, err = fd.Write(buf)
	if err != nil {
		log.Errorf("写入 stackdump 失败: %v", err)
		log.Warnf("无法保存 stackdump. 将直接打印\n %s", buf)
		return
	}
	log.Infof("stackdump 已保存至 %s", fileName)
}
