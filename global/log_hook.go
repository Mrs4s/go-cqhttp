package global

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sync"

	"github.com/sirupsen/logrus"
)

// LocalHook logrus本地钩子
type LocalHook struct {
	lock      *sync.Mutex
	levels    []logrus.Level   // hook级别
	formatter logrus.Formatter // 格式
	path      string           // 写入path
	writer    io.Writer        // io
}

// Levels ref: logrus/hooks.go impl Hook interface
func (hook *LocalHook) Levels() []logrus.Level {
	if len(hook.levels) == 0 {
		return logrus.AllLevels
	}
	return hook.levels
}

func (hook *LocalHook) ioWrite(entry *logrus.Entry) error {
	log, err := hook.formatter.Format(entry)
	if err != nil {
		return err
	}

	_, err = hook.writer.Write(log)
	if err != nil {
		return err
	}
	return nil
}

func (hook *LocalHook) pathWrite(entry *logrus.Entry) error {
	dir := filepath.Dir(hook.path)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}

	fd, err := os.OpenFile(hook.path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer fd.Close()

	log, err := hook.formatter.Format(entry)
	if err != nil {
		return err
	}

	_, err = fd.Write(log)
	return err
}

// Fire ref: logrus/hooks.go impl Hook interface
func (hook *LocalHook) Fire(entry *logrus.Entry) error {
	hook.lock.Lock()
	defer hook.lock.Unlock()

	if hook.writer != nil {
		return hook.ioWrite(entry)
	}

	if hook.path != "" {
		return hook.pathWrite(entry)
	}

	return nil
}

// SetFormatter 设置日志格式
func (hook *LocalHook) SetFormatter(formatter logrus.Formatter) {
	hook.lock.Lock()
	defer hook.lock.Unlock()

	if formatter == nil {
		// 用默认的
		formatter = &logrus.TextFormatter{DisableColors: true}
	} else {
		switch f := formatter.(type) {
		case *logrus.TextFormatter:
			textFormatter := f
			textFormatter.DisableColors = true
		default:
			// todo
		}
	}
	logrus.SetFormatter(formatter)
	hook.formatter = formatter
}

// SetWriter 设置Writer
func (hook *LocalHook) SetWriter(writer io.Writer) {
	hook.lock.Lock()
	defer hook.lock.Unlock()
	hook.writer = writer
}

// SetPath 设置日志写入路径
func (hook *LocalHook) SetPath(path string) {
	hook.lock.Lock()
	defer hook.lock.Unlock()
	hook.path = path
}

// NewLocalHook 初始化本地日志钩子实现
func NewLocalHook(args interface{}, formatter logrus.Formatter, levels ...logrus.Level) *LocalHook {
	hook := &LocalHook{
		lock: new(sync.Mutex),
	}
	hook.SetFormatter(formatter)
	hook.levels = append(hook.levels, levels...)

	switch arg := args.(type) {
	case string:
		hook.SetPath(arg)
	case io.Writer:
		hook.SetWriter(arg)
	default:
		panic(fmt.Sprintf("unsupported type: %v", reflect.TypeOf(args)))
	}

	return hook
}

// GetLogLevel 获取日志等级
//
// 可能的值有
//
// "trace","debug","info","warn","warn","error"
func GetLogLevel(level string) []logrus.Level {
	switch level {
	case "trace":
		return []logrus.Level{
			logrus.TraceLevel, logrus.DebugLevel,
			logrus.InfoLevel, logrus.WarnLevel, logrus.ErrorLevel,
			logrus.FatalLevel, logrus.PanicLevel,
		}
	case "debug":
		return []logrus.Level{
			logrus.DebugLevel, logrus.InfoLevel,
			logrus.WarnLevel, logrus.ErrorLevel,
			logrus.FatalLevel, logrus.PanicLevel,
		}
	case "info":
		return []logrus.Level{
			logrus.InfoLevel, logrus.WarnLevel,
			logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel,
		}
	case "warn":
		return []logrus.Level{
			logrus.WarnLevel, logrus.ErrorLevel,
			logrus.FatalLevel, logrus.PanicLevel,
		}
	case "error":
		return []logrus.Level{
			logrus.ErrorLevel, logrus.FatalLevel,
			logrus.PanicLevel,
		}
	default:
		return []logrus.Level{
			logrus.InfoLevel, logrus.WarnLevel,
			logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel,
		}
	}
}
