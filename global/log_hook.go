package global

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"github.com/mattn/go-colorable"
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

	fd, err := os.OpenFile(hook.path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o666)
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
func (hook *LocalHook) SetFormatter(consoleFormatter, fileFormatter logrus.Formatter) {
	hook.lock.Lock()
	defer hook.lock.Unlock()

	// 支持处理windows平台的console色彩
	logrus.SetOutput(colorable.NewColorableStdout())
	// 用于在console写出
	logrus.SetFormatter(consoleFormatter)
	// 用于写入文件
	hook.formatter = fileFormatter
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
func NewLocalHook(args interface{}, consoleFormatter, fileFormatter logrus.Formatter, levels ...logrus.Level) *LocalHook {
	hook := &LocalHook{
		lock: new(sync.Mutex),
	}
	hook.SetFormatter(consoleFormatter, fileFormatter)
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

// LogFormat specialize for go-cqhttp
type LogFormat struct {
	EnableColor bool
}

// Format implements logrus.Formatter
func (f LogFormat) Format(entry *logrus.Entry) ([]byte, error) {
	buf := NewBuffer()
	defer PutBuffer(buf)

	if f.EnableColor {
		buf.WriteString(GetLogLevelColorCode(entry.Level))
	}

	buf.WriteByte('[')
	buf.WriteString(entry.Time.Format("2006-01-02 15:04:05"))
	buf.WriteString("] [")
	buf.WriteString(strings.ToUpper(entry.Level.String()))
	buf.WriteString("]: ")
	buf.WriteString(entry.Message)
	buf.WriteString(" \n")

	if f.EnableColor {
		buf.WriteString(colorReset)
	}

	ret := make([]byte, len(buf.Bytes()))
	copy(ret, buf.Bytes()) // copy buffer
	return ret, nil
}

const (
	colorCodePanic = "\x1b[1;31m" // color.Style{color.Bold, color.Red}.String()
	colorCodeFatal = "\x1b[1;31m" // color.Style{color.Bold, color.Red}.String()
	colorCodeError = "\x1b[31m"   // color.Style{color.Red}.String()
	colorCodeWarn  = "\x1b[33m"   // color.Style{color.Yellow}.String()
	colorCodeInfo  = "\x1b[37m"   // color.Style{color.White}.String()
	colorCodeDebug = "\x1b[32m"   // color.Style{color.Green}.String()
	colorCodeTrace = "\x1b[36m"   // color.Style{color.Cyan}.String()
	colorReset     = "\x1b[0m"
)

// GetLogLevelColorCode 获取日志等级对应色彩code
func GetLogLevelColorCode(level logrus.Level) string {
	switch level {
	case logrus.PanicLevel:
		return colorCodePanic
	case logrus.FatalLevel:
		return colorCodeFatal
	case logrus.ErrorLevel:
		return colorCodeError
	case logrus.WarnLevel:
		return colorCodeWarn
	case logrus.InfoLevel:
		return colorCodeInfo
	case logrus.DebugLevel:
		return colorCodeDebug
	case logrus.TraceLevel:
		return colorCodeTrace

	default:
		return colorCodeInfo
	}
}
