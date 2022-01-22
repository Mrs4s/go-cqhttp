package loghook

import "github.com/Mrs4s/go-cqhttp/cmd/iris_admin/utils/common"

// WebLogWriter 日志的Witer结构体
type WebLogWriter struct {
	log common.FixedList
}

// NewWebLogWriter 创建io.Writer 的实现
func NewWebLogWriter() *WebLogWriter {
	return &WebLogWriter{
		log: common.NewFixedList(100),
	}
}

// Write 实现 io.Writer 的方法
func (w *WebLogWriter) Write(p []byte) (n int, err error) {
	w.log.Add(p)
	return 0, err
}

// Read 读取日志
func (w *WebLogWriter) Read() string {
	data := w.log.Data()
	var str string
	for _, v := range data {
		str = str + "</br>" + string(v.([]byte))
	}
	return str
}
