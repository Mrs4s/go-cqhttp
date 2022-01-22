// Package jump 信息跳转
package jump

import (
	"github.com/GoAdminGroup/go-admin/template/types"
	"github.com/kataras/iris/v12"

	"github.com/Mrs4s/go-cqhttp/cmd/iris_admin/utils/common"
)

// Error 错误信息提示+跳转
func Error(data common.Msg) types.Panel {
	var path = "html/error.tmpl"
	tmp, err := common.HTMLFilesHandler(data, path)
	if err != nil {
		panic(err)
		// return Error(Msg{
		//	Msg:  err.Error(),
		//	URL:  data.URL,
		//	Wait: data.Wait,
		// })
	}
	return types.Panel{
		Title:   "跳转提示",
		Content: tmp,
	}
}

// ErrorForIris 错误信息提示+跳转 iris适配
func ErrorForIris(ctx iris.Context, data common.Msg) {
	var path = "html/error.tmpl"
	tmp, err := common.HTMLFilesHandlerString(data, path)
	if err != nil {
		panic(err)
	}
	_, _ = ctx.HTML(tmp)
}
