package server

import (
	"encoding/json"
	"github.com/Mrs4s/go-cqhttp/coolq"
	"github.com/Mrs4s/go-cqhttp/global"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"html/template"
	"net/http"
	"os"
	"strings"
	log "github.com/sirupsen/logrus"
	"time"
)

type webServer struct {
	engine *gin.Engine
	bot    *coolq.CQBot
}

var WebServer = &webServer{}

func (s *webServer) Run(addr string, bot *coolq.CQBot) {
	//r.Run() // listen and serve on 0.0.0.0:8080
	gin.SetMode(gin.ReleaseMode)
	s.engine = gin.New()
	s.bot = bot
	s.engine.LoadHTMLGlob("html/*")
	//静态资源
	s.engine.StaticFS("/assets", http.Dir("./html/assets"))
	//s.engine.StaticFile("/favicon.ico", "./html/favicon.ico")
	// 无参数
	s.engine.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/admin/index")
	})
	//通用路由
	s.engine.Any("/admin/:action", s.admin)
	s.engine.Use(func(c *gin.Context) {
		if c.Request.Method != "GET" && c.Request.Method != "POST" {
			log.Warnf("已拒绝客户端 %v 的请求: 方法错误", c.Request.RemoteAddr)
			c.Status(404)
			return
		}
		if c.Request.Method == "POST" && strings.Contains(c.Request.Header.Get("Content-Type"), "application/json") {
			d, err := c.GetRawData()
			if err != nil {
				log.Warnf("获取请求 %v 的Body时出现错误: %v", c.Request.RequestURI, err)
				c.Status(400)
				return
			}
			if !gjson.ValidBytes(d) {
				log.Warnf("已拒绝客户端 %v 的请求: 非法Json", c.Request.RemoteAddr)
				c.Status(400)
				return
			}
			c.Set("json_body", gjson.ParseBytes(d))
		}
		c.Next()
	})
	go func() {
		log.Infof("CQ HTTP 服务器已启动: %v", addr)
		err := s.engine.Run(addr)
		if err != nil {
			log.Error(err)
			log.Infof("请检查端口是否被占用.")
			time.Sleep(time.Second * 5)
			os.Exit(1)
		}
	}()
}
func (s *webServer) admin(c *gin.Context) {
	action := c.Param("action")
	log.Debugf("HTTPServer接收到API调用: %v", action)
	if f, ok := httpuri[action]; ok {
		f(s, c)
	} else {
		c.JSON(200, coolq.Failed(404))
	}
}
var httpuri = map[string]func(s *webServer, c *gin.Context){
	"index": func(s *webServer, c *gin.Context) {
		s.AdminIndex(c)
	},
	"test": func(s *webServer, c *gin.Context) {
		s.test(c)
	},
}

func  (s *webServer)AdminIndex(c *gin.Context) {
	conf := global.Load("config.json")
	uin:= conf.Uin
	passwd:=conf.Password
	confJson ,_ := json.Marshal(conf)
	c.HTML(http.StatusOK, "index.html", gin.H{
		"uin":  uin,
		"passwd": passwd,
		"conf":template.HTML(string(confJson)),
	})
}

func  (s *webServer)test(c *gin.Context) {
	h :=*HttpServer
	h.CanSendImage(c)
}
