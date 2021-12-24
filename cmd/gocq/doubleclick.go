package gocq

import (
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/Mrs4s/go-cqhttp/global/terminal"
	"github.com/Mrs4s/go-cqhttp/internal/base"
)

// CheckDoubleClick 检查双击启动
func CheckDoubleClick() {
	if !base.FastStart && terminal.RunningByDoubleClick() {
		err := terminal.NoMoreDoubleClick()
		if err != nil {
			log.Errorf("遇到错误: %v", err)
			time.Sleep(time.Second * 5)
		}
		os.Exit(0)
	}
}
