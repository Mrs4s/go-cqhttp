//+build !windows

package global

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func SetupMainSignalHandler() <-chan struct{} {
	mainOnce.Do(func() {
		mc := make(chan os.Signal, 2)
		closeOnce := sync.Once{}
		signal.Notify(mc, os.Interrupt, syscall.SIGTERM, syscall.SIGUSR1)
		go func() {
			for {
				s := <-mc
				switch s {
				case os.Interrupt, syscall.SIGTERM:
					closeOnce.Do(func() {
						close(mc)
					})
				case syscall.SIGUSR1:
					dumpStack()
				}
			}
		}()

		mainStopCh = make(chan struct{})
	})
	return mainStopCh
}
