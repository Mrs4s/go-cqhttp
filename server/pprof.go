package server

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"github.com/Mrs4s/go-cqhttp/global/config"
)

// RunPprofServer 启动 pprof 性能分析服务器
func RunPprofServer(conf *config.PprofServer) {
	if conf.Disabled {
		return
	}
	engine := gin.New()
	addr := fmt.Sprintf("%s:%d", conf.Host, conf.Port)
	pprof.Register(engine)
	go func() {
		log.Infof("pprof debug 服务器已启动: %v/debug/pprof", addr)
		log.Warnf("警告: pprof 服务不支持鉴权, 请不要运行在公网.")
		if err := engine.Run(addr); err != nil && err != http.ErrServerClosed {
			log.Error(err)
			log.Infof("pprof 服务启动失败, 请检查端口是否被占用.")
			log.Warnf("将在五秒后退出.")
			time.Sleep(time.Second * 5)
			os.Exit(1)
		}
	}()
}
