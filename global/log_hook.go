package global

import (
	"io"
	"strings"

	"github.com/mattn/go-colorable"

	"github.com/Mrs4s/go-cqhttp/internal/log"
)

// LocalHook logrus本地钩子
type LocalHook struct {
	level     log.Level
	formatter log.Formatter // 格式
	writer    io.Writer     // io
}

// Level impl Hook interface
func (hook *LocalHook) Level() log.Level {
	return hook.level
}

func (hook *LocalHook) ioWrite(entry *log.Entry) {
	_, _ = hook.writer.Write(hook.formatter.Format(entry))
}

// Fire ref: logrus/hooks.go impl Hook interface
func (hook *LocalHook) Fire(entry *log.Entry) {
	if hook.writer != nil {
		hook.ioWrite(entry)
	}
}

// SetFormatter 设置日志格式
func (hook *LocalHook) SetFormatter(consoleFormatter, fileFormatter log.Formatter) {
	// 支持处理windows平台的console色彩
	log.SetOutput(colorable.NewColorableStdout())
	// 用于在console写出
	log.SetFormatter(consoleFormatter)
	// 用于写入文件
	hook.formatter = fileFormatter
}

// NewLocalHook 初始化本地日志钩子实现
func NewLocalHook(local io.Writer, consoleFormatter, fileFormatter log.Formatter, level log.Level) *LocalHook {
	hook := new(LocalHook)
	hook.SetFormatter(consoleFormatter, fileFormatter)
	hook.level = level
	hook.writer = local
	return hook
}

// GetLogLevel 获取日志等级
//
// 可能的值有
//
// "debug","info","warn","warn","error"
func GetLogLevel(level string) log.Level {
	switch level {
	case "debug":
		return log.DebugLevel
	case "info":
		return log.InfoLevel
	case "warn":
		return log.WarnLevel
	case "error":
		return log.ErrorLevel
	default:
		return log.InfoLevel
	}
}

// LogFormat specialize for go-cqhttp
type LogFormat struct {
	EnableColor bool
}

// Format implements logrus.Formatter
func (f LogFormat) Format(entry *log.Entry) []byte {
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

	ret := append([]byte(nil), buf.Bytes()...) // copy buffer
	return ret
}

const (
	colorCodeFatal = "\x1b[1;31m" // color.Style{color.Bold, color.Red}.String()
	colorCodeError = "\x1b[31m"   // color.Style{color.Red}.String()
	colorCodeWarn  = "\x1b[33m"   // color.Style{color.Yellow}.String()
	colorCodeInfo  = "\x1b[37m"   // color.Style{color.White}.String()
	colorCodeDebug = "\x1b[32m"   // color.Style{color.Green}.String()
	colorReset     = "\x1b[0m"
)

// GetLogLevelColorCode 获取日志等级对应色彩code
func GetLogLevelColorCode(level log.Level) string {
	switch level {
	case log.FatalLevel:
		return colorCodeFatal
	case log.ErrorLevel:
		return colorCodeError
	case log.WarnLevel:
		return colorCodeWarn
	case log.InfoLevel:
		return colorCodeInfo
	case log.DebugLevel:
		return colorCodeDebug

	default:
		return colorCodeInfo
	}
}
