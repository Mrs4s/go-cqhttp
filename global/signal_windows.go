//go:build windows
// +build windows

package global

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Microsoft/go-winio"
	log "github.com/sirupsen/logrus"
)

var validTasks = map[string]func(){
	"dumpstack": dumpStack,
}

// SetupMainSignalHandler is for main to use at last
func SetupMainSignalHandler() <-chan struct{} {
	mainOnce.Do(func() {
		// for stack trace collecting on windows
		pipeName := fmt.Sprintf(`\\.\pipe\go-cqhttp-%d`, os.Getpid())
		pipe, err := winio.ListenPipe(pipeName, &winio.PipeConfig{})
		if err != nil {
			log.Errorf("创建 named pipe 失败. 将无法使用 dumpstack 功能: %v", err)
		} else {
			maxTaskLen := 0
			for t := range validTasks {
				if l := len(t); l > maxTaskLen {
					maxTaskLen = l
				}
			}
			go func() {
				for {
					c, err := pipe.Accept()
					if err != nil {
						if errors.Is(err, net.ErrClosed) || strings.Contains(err.Error(), "closed") {
							return
						}
						log.Errorf("accept named pipe 失败: %v", err)
						continue
					}
					go func() {
						defer c.Close()
						_ = c.SetReadDeadline(time.Now().Add(5 * time.Second))
						buf := make([]byte, maxTaskLen)
						n, err := c.Read(buf)
						if err != nil {
							log.Errorf("读取 named pipe 失败: %v", err)
							return
						}
						cmd := string(buf[:n])
						if task, ok := validTasks[cmd]; ok {
							task()
							return
						}
						log.Warnf("named pipe 读取到未知指令: %q", cmd)
					}()
				}
			}()
		}
		// setup the main stop channel
		mainStopCh = make(chan struct{})
		mc := make(chan os.Signal, 2)
		closeOnce := sync.Once{}
		signal.Notify(mc, os.Interrupt, syscall.SIGTERM)
		go func() {
			for {
				switch <-mc {
				case os.Interrupt, syscall.SIGTERM:
					closeOnce.Do(func() {
						close(mainStopCh)
						if pipe != nil {
							_ = pipe.Close()
						}
					})
				}
			}
		}()
	})
	return mainStopCh
}
