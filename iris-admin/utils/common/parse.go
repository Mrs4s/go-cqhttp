package common

import (
	"bytes"
	"github.com/GoAdminGroup/go-admin/template"
	template2 "html/template"
)

type Msg struct {
	Msg  string
	Url  string
	Wait int64
}

// HTMLFiles inject the route and corresponding handler which returns the panel content of given html files path
// to the web framework.
func HtmlFilesHandler(data Msg, files ...string) (template2.HTML, error) {
	if data.Wait == 0 {
		data.Wait = 3
	}
	cbuf := new(bytes.Buffer)
	t, err := template2.ParseFS(GetHtmlFs(), files...)
	if err != nil {
		return "", err
	} else if err := t.Execute(cbuf, data); err != nil {
		return "", err
	}
	return template.HTML(cbuf.String()), err
}
