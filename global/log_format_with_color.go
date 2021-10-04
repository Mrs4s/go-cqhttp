//go:build with_color
// +build with_color

package global

import (
	"fmt"
	"strings"

	"github.com/gookit/color"
	"github.com/sirupsen/logrus"
)

// LogFormat specialize for go-cqhttp
type LogFormat struct{}

// Format implements logrus.Formatter
func (f LogFormat) Format(entry *logrus.Entry) ([]byte, error) {
	buf := NewBuffer()
	defer PutBuffer(buf)

	buf.WriteString(getLogLevelColorCode(entry.Level))

	buf.WriteByte('[')
	buf.WriteString(entry.Time.Format("2006-01-02 15:04:05"))
	buf.WriteString("] [")
	buf.WriteString(strings.ToUpper(entry.Level.String()))
	buf.WriteString("]: ")
	buf.WriteString(entry.Message)
	buf.WriteString(" \n")

	buf.WriteString(color.ResetSet)

	ret := append([]byte(nil), buf.Bytes()...) // copy buffer
	return ret, nil
}

var (
	colorCodePanic = fmt.Sprintf(color.SettingTpl, color.Style{color.Bold, color.Red}.String())
	colorCodeFatal = fmt.Sprintf(color.SettingTpl, color.Style{color.Bold, color.Red}.String())
	colorCodeError = fmt.Sprintf(color.SettingTpl, color.Style{color.Red}.String())
	colorCodeWarn  = fmt.Sprintf(color.SettingTpl, color.Style{color.Yellow}.String())
	colorCodeInfo  = fmt.Sprintf(color.SettingTpl, color.Style{color.Green}.String())
	colorCodeDebug = fmt.Sprintf(color.SettingTpl, color.Style{color.White}.String())
	colorCodeTrace = fmt.Sprintf(color.SettingTpl, color.Style{color.Cyan}.String())
)

// getLogLevelColorCode 获取日志等级对应色彩code
func getLogLevelColorCode(level logrus.Level) string {
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
