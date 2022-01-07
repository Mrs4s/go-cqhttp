package jump

import (
	"github.com/GoAdminGroup/go-admin/template/types"
	"github.com/Mrs4s/go-cqhttp/iris-admin/utils/common"
	"github.com/kataras/iris/v12"
)

func JumpSuccess(data common.Msg) types.Panel {
	var path = "html/success.tmpl"
	tmp, err := common.HtmlFilesHandler(data, path)
	if err != nil {
		panic(err)
		//return JumpError(Msg{
		//	Msg: err.Error(),
		//	Url: data.Url,
		//	Wait: data.Wait,
		//})
	}
	return types.Panel{
		Title:   "跳转提示",
		Content: tmp,
	}
}

func JumpSuccessForIris(ctx iris.Context, data common.Msg) {
	var path = "html/success.tmpl"
	tmp, err := common.HtmlFilesHandlerString(data, path)
	if err != nil {
		panic(err)
	}
	ctx.HTML(tmp)
}
