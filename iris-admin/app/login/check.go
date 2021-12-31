package login

import (
	"github.com/GoAdminGroup/go-admin/engine"
	models2 "github.com/GoAdminGroup/go-admin/plugins/admin/models"
	"github.com/GoAdminGroup/go-admin/template/types"
	"github.com/Mrs4s/go-cqhttp/iris-admin/utils/common"
	"github.com/Mrs4s/go-cqhttp/iris-admin/utils/jump"
	"github.com/kataras/iris/v12"
	log "github.com/sirupsen/logrus"
)

//验证 QQ是否已经登录
func CheckQQlogin(ctx iris.Context) (types.Panel, error) {
	var user models2.UserModel
	var ok bool
	user, ok = engine.User(ctx)
	if !ok {
		return jump.JumpError(common.Msg{
			Msg:  "获取登录信息失败",
			Url:  "/admin/",
			Wait: 3,
		}), nil
	}
	log.Debug(user)
	return jump.JumpSuccess(common.Msg{
		Msg:  "qq已成功登陆",
		Url:  "/admin/qq/info",
		Wait: 3,
	}), nil
}
