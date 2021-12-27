package modules

import (
	"fmt"
	"github.com/Mrs4s/MiraiGo/client"
	"github.com/Mrs4s/go-cqhttp/global"
	log "github.com/sirupsen/logrus"
	"os"
	"path"
	"time"
)

// Logger TODO 更合理的Logger
type Logger struct{}

func (l *Logger) Error(c *client.QQClient, msg string, args ...interface{}) {
	log.Error("Protocol -> " + fmt.Sprintf(msg, args...))
}

func (l *Logger) Warning(c *client.QQClient, msg string, args ...interface{}) {
	//log.Warnf(msg, args...)
}

func (l *Logger) Info(c *client.QQClient, msg string, args ...interface{}) {
	log.Info("Protocol -> " + fmt.Sprintf(msg, args...))
}

func (l *Logger) Debug(c *client.QQClient, msg string, args ...interface{}) {
	log.Debug("Protocol -> " + fmt.Sprintf(msg, args...))
}

func (l *Logger) Trace(c *client.QQClient, msg string, args ...interface{}) {
	//log.Tracef(msg, args...)
}

func (l *Logger) Dump(c *client.QQClient, dump []byte, msg string, args ...interface{}) {
	if !global.PathExists(global.DumpsPath) {
		_ = os.MkdirAll(global.DumpsPath, 0o755)
	}
	dumpFile := path.Join(global.DumpsPath, fmt.Sprintf("%v.dump", time.Now().Unix()))
	log.Errorf("出现错误 %v. 详细信息已转储至文件 %v 请连同日志提交给开发者处理", fmt.Sprintf(msg, args...), dumpFile)
	_ = os.WriteFile(dumpFile, dump, 0o644)
}
