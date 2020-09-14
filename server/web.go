package server

import (
	"fmt"
	"github.com/Mrs4s/go-cqhttp/coolq"
	"github.com/Mrs4s/go-cqhttp/global"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/cpu"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"html/template"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"
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
	//func 函数映射
	s.engine.SetFuncMap(template.FuncMap{
		"getYear":       getYear,
		"formatAsDate":  formatAsDate,
		"getConf":       getConf,
		"getDate":       getDate,
		"getavator":     getavator,
		"getServerInfo": getServerInfo,
		"formatFileSize": formatFileSize,
	})
	s.engine.LoadHTMLGlob("template/html/**/*")
	//静态资源
	s.engine.Static("/assets", "./template/assets")
	//s.engine.StaticFS("/assets", http.Dir("html/assets"))
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

func (s *webServer) AdminIndex(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{})
}

func (s *webServer) test(c *gin.Context) {
	//h :=*HttpServer
	//h.CanSendImage(c)
	println(os.Args[0])
}
func formatAsDate(t time.Time) string {
	year, month, day := t.Date()
	return fmt.Sprintf("%d%02d/%02d", year, month, day)
}
func formatAsYear(t time.Time) string {
	year, _, _ := t.Date()
	return fmt.Sprintf("%d", year)
}
func getYear() string {
	t := time.Now()
	year, _, _ := t.Date()
	return fmt.Sprintf("%d", year)
}

func getDate() string {
	t := time.Now()
	year, month, day := t.Date()
	return fmt.Sprintf("%d-%02d-%02d", year, month, day)
}

func getConf() *global.JsonConfig {
	conf := global.Load("config.json")
	return conf
}

func getavator() string {
	Uuid := uuid.New().String()
	grav_url := "https://www.gravatar.com/avatar/" + Uuid
	return grav_url
}

type info struct {
	Root       string
	Version    string
	Hostname   string
	Interfaces interface{}
	Goarch     string
	Goos       string
	VirtualMemory *mem.VirtualMemoryStat
	Sys uint64
}

func getServerInfo() *info {
	root := runtime.GOROOT()          // GO 路径
	version := runtime.Version()      //GO 版本信息
	hostname, _ := os.Hostname()      //获得PC名
	interfaces, _ := net.Interfaces() //获得网卡信息
	goarch := runtime.GOARCH          //系统构架 386、amd64
	goos := runtime.GOOS              //系统版本 windows
	Info := &info{
		Root:       root,
		Version:    version,
		Hostname:   hostname,
		Interfaces: interfaces,
		Goarch:     goarch,
		Goos:       goos,
	}

	v, _ := mem.VirtualMemory()
	Info.VirtualMemory=v
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	Info.Sys=ms.Sys

	//log.Printf("Alloc:%d(bytes) HeapIdle:%d(bytes) HeapReleased:%d(bytes)", ms.Alloc, ms.HeapIdle, ms.HeapReleased)
	return Info
}



// 字节的单位转换 保留两位小数
func formatFileSize(fileSize uint64) (size string) {
	if fileSize < 1024 {
		//return strconv.FormatInt(fileSize, 10) + "B"
		return fmt.Sprintf("%.2fB", float64(fileSize)/float64(1))
	} else if fileSize < (1024 * 1024) {
		return fmt.Sprintf("%.2fKB", float64(fileSize)/float64(1024))
	} else if fileSize < (1024 * 1024 * 1024) {
		return fmt.Sprintf("%.2fMB", float64(fileSize)/float64(1024*1024))
	} else if fileSize < (1024 * 1024 * 1024 * 1024) {
		return fmt.Sprintf("%.2fGB", float64(fileSize)/float64(1024*1024*1024))
	} else if fileSize < (1024 * 1024 * 1024 * 1024 * 1024) {
		return fmt.Sprintf("%.2fTB", float64(fileSize)/float64(1024*1024*1024*1024))
	} else { //if fileSize < (1024 * 1024 * 1024 * 1024 * 1024 * 1024)
		return fmt.Sprintf("%.2fEB", float64(fileSize)/float64(1024*1024*1024*1024*1024))
	}
}