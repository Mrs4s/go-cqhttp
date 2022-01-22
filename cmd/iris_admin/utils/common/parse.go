package common

import (
	"bytes"
	template2 "html/template"

	"github.com/GoAdminGroup/go-admin/template"
)

// Msg 消息结构体
type Msg struct {
	Msg  string // 错误/成功 信息
	URL  string // 跳转地址
	Wait int64  // 跳转等待时间 秒
}

// HTMLFilesHandler 从路径读取模板
func HTMLFilesHandler(data Msg, files ...string) (template2.HTML, error) {
	if data.Wait == 0 {
		data.Wait = 3
	}
	cbuf := new(bytes.Buffer)
	t, err := template2.ParseFS(GetStaticFs(), files...)
	if err != nil {
		return "", err
	} else if err := t.Execute(cbuf, data); err != nil {
		return "", err
	}
	return template.HTML(cbuf.String()), err
}

// HTMLFilesHandlerString 从路径读取模板字符串
func HTMLFilesHandlerString(data Msg, files ...string) (string, error) {
	if data.Wait == 0 {
		data.Wait = 3
	}
	cbuf := new(bytes.Buffer)
	t, err := template2.ParseFS(GetStaticFs(), files...)
	if err != nil {
		return "", err
	} else if err := t.Execute(cbuf, data); err != nil {
		return "", err
	}
	return cbuf.String(), err
}
