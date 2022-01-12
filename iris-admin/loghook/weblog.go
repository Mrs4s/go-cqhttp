package loghook

import "github.com/Mrs4s/go-cqhttp/iris-admin/utils/common"

type WebLogWriter struct {
	log common.FixedList
}

func NewWebLogWriter() *WebLogWriter {
	return &WebLogWriter{
		log: common.NewFixedList(100),
	}
}

func (w *WebLogWriter) Write(p []byte) (n int, err error) {
	w.log.Add(p)
	return 0, err
}

func (w *WebLogWriter) Read() string {
	data := w.log.Data()
	var str string
	for _, v := range data {
		str = str + "</br>" + string(v.([]byte))
	}
	return str
}
