//+build windows

package global

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Microsoft/go-winio"
	log "github.com/sirupsen/logrus"
)

var (
	validTasks = []string{
		"dumpstack",
	}
)

func SetupMainSignalHandler() <-chan struct{} {
	mainOnce.Do(func() {
		// for stack trace collecting on windows
		pipeName := fmt.Sprintf(`\\.\pipe\go-cqhttp-%d`, os.Getpid())
		pipe, err := winio.ListenPipe(pipeName, &winio.PipeConfig{})
		if err != nil {
			log.Error("创建 named pipe 失败. 将无法使用 dumpstack 功能")
		} else {
			maxTaskLen := 0
			for i := range validTasks {
				if l := len(validTasks[i]); l > maxTaskLen {
					maxTaskLen = l
				}
			}
			go func() {
				for {
					c, err := pipe.Accept()
					if err != nil {
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
						switch cmd {
						case "dumpstack":
							dumpStack()
						default:
							log.Warnf("named pipe 读取到未知指令: %q", cmd)
						}
					}()
				}
			}()
		}

		mc := make(chan os.Signal, 2)
		closeOnce := sync.Once{}
		signal.Notify(mc, os.Interrupt, syscall.SIGTERM)
		go func() {
			for {
				s := <-mc
				switch s {
				case os.Interrupt, syscall.SIGTERM:
					closeOnce.Do(func() {
						close(mc)
						if pipe != nil {
							_ = pipe.Close()
						}
					})
				}
			}
		}()

		mainStopCh = make(chan struct{})
	})
	return mainStopCh
}
