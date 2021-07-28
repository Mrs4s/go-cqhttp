//go:build !windows
// +build !windows

package global

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// SetupMainSignalHandler is for main to use at last
func SetupMainSignalHandler() <-chan struct{} {
	mainOnce.Do(func() {
		mainStopCh = make(chan struct{})
		mc := make(chan os.Signal, 3)
		closeOnce := sync.Once{}
		signal.Notify(mc, os.Interrupt, syscall.SIGTERM, syscall.SIGUSR1)
		go func() {
			for {
				switch <-mc {
				case os.Interrupt, syscall.SIGTERM:
					closeOnce.Do(func() {
						close(mainStopCh)
					})
				case syscall.SIGUSR1:
					dumpStack()
				}
			}
		}()
	})
	return mainStopCh
}
