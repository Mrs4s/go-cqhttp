package gocq

import (
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/Mrs4s/go-cqhttp/global"
	"github.com/Mrs4s/go-cqhttp/internal/base"
)

// PrintBanner 打印 gocq 欢迎信息
func PrintBanner() {
	if (base.Account.Uin == 0 || (base.Account.Password == "" && !base.Account.Encrypt)) && !global.PathExists("session.token") {
		log.Warn("账号密码未配置, 将使用二维码登录.")
		if !base.FastStart {
			log.Warn("将在 5秒 后继续.")
			time.Sleep(time.Second * 5)
		}
	}

	log.Info("当前版本:", base.Version)
	if base.Debug {
		log.SetLevel(log.DebugLevel)
		log.SetReportCaller(true)
		log.Warnf("已开启Debug模式.")
		log.Debugf("开发交流群: 192548878")
	}
	log.Info("用户交流群: 721829413")
}
