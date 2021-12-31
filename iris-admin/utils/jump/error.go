package jump

import (
	"github.com/GoAdminGroup/go-admin/template/types"
	"github.com/Mrs4s/go-cqhttp/iris-admin/utils/common"
)

func JumpError(data common.Msg) types.Panel {
	var path = "html/error.tmpl"
	tmp, err := common.HtmlFilesHandler(data, path)
	if err != nil {
		panic(err)
		//return JumpError(Msg{
		//	Msg:  err.Error(),
		//	Url:  data.Url,
		//	Wait: data.Wait,
		//})
	}
	return types.Panel{
		Title: "跳转提示",
		Content: tmp,
	}
}
