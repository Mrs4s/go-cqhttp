package jump

import (
	"github.com/GoAdminGroup/go-admin/template/types"
	"github.com/kataras/iris/v12"

	"github.com/Mrs4s/go-cqhttp/cmd/iris_admin/utils/common"
)

// Success 成功信息提示+跳转
func Success(data common.Msg) types.Panel {
	var path = "html/success.tmpl"
	tmp, err := common.HTMLFilesHandler(data, path)
	if err != nil {
		panic(err)
		// return Error(Msg{
		//	Msg: err.Error(),
		//	URL: data.URL,
		//	Wait: data.Wait,
		// })
	}
	return types.Panel{
		Title:   "跳转提示",
		Content: tmp,
	}
}

// SuccessForIris 成功信息提示+跳转 iris适配
func SuccessForIris(ctx iris.Context, data common.Msg) {
	var path = "html/success.tmpl"
	tmp, err := common.HTMLFilesHandlerString(data, path)
	if err != nil {
		panic(err)
	}
	_, _ = ctx.HTML(tmp)
}
