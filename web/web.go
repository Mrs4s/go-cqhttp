package web

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/Mrs4s/go-cqhttp/coolq"
	"github.com/Mrs4s/go-cqhttp/global"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	//"github.com/shirou/gopsutil/cpu"

	//"github.com/shirou/gopsutil/mem"
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
	gin.SetMode(gin.ReleaseMode)
	s.engine = gin.New()
	s.bot = bot //外部引入 bot对象，用于操作bot
	//func 函数映射 全局模板可用
	s.engine.SetFuncMap(template.FuncMap{
		"getYear":        GetYear,
		"formatAsDate":   FormatAsDate,
		"getConf":        GetConf,
		"getDate":        GetDate,
		"getavator":      Getavator,
		"getServerInfo":  GetServerInfo,
		"formatFileSize": FormatFileSize,
	})
	s.engine.LoadHTMLGlob("template/html/**/*")
	//静态资源
	s.engine.Static("/assets", "./template/assets")
	//s.engine.StaticFile("/favicon.ico", "./html/favicon.ico")
	// 自动转跳到 admin/index
	s.engine.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/index/login")
	})
	//通用路由
	s.engine.Any("/admin/:action",AuthMiddleWare(), s.admin)
	s.engine.Any("/index/:action",s.index)
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
	log.Debugf("WebServer接收到cgi调用: %v", action)
	if f, ok := HttpuriAdmin[action]; ok {
		f(s, c)
	} else {
		c.JSON(200, coolq.Failed(404))
	}
}

func (s *webServer) index(c *gin.Context) {
	action := c.Param("action")
	log.Debugf("WebServer接收到cgi调用: %v", action)
	if f, ok := HttpuriIndex[action]; ok {
		f(s, c)
	} else {
		c.JSON(200, coolq.Failed(404))
	}
}

//格式化年月日
func FormatAsDate(t time.Time) string {
	year, month, day := t.Date()
	return fmt.Sprintf("%d%02d/%02d", year, month, day)
}

// 获取年份
func GetYear() string {
	t := time.Now()
	year, _, _ := t.Date()
	return fmt.Sprintf("%d", year)
}

// 获取当前年月日
func GetDate() string {
	t := time.Now()
	year, month, day := t.Date()
	return fmt.Sprintf("%d-%02d-%02d", year, month, day)
}

// 获取当前配置文件信息
func GetConf() *global.JsonConfig {
	conf := global.Load("config.json")
	return conf
}

// 随机获取一个头像
func Getavator() string {
	Uuid := uuid.New().String()
	grav_url := "https://www.gravatar.com/avatar/" + Uuid
	return grav_url
}

type info struct {
	Root          string
	Version       string
	Hostname      string
	Interfaces    interface{}
	Goarch        string
	Goos          string
	//VirtualMemory *mem.VirtualMemoryStat
	Sys           uint64
	CpuInfoStat   struct {
		Count   int
		Percent []float64
	}
}

func GetServerInfo() *info {
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

	//v, _ := mem.VirtualMemory()
	//Info.VirtualMemory = v
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	Info.Sys = ms.Sys
	//Info.CpuInfoStat.Count, _ = cpu.Counts(true)
	//Info.CpuInfoStat.Percent, _ = cpu.Percent(0, true)
	return Info
}

// 字节的单位转换 保留两位小数
func FormatFileSize(fileSize uint64) (size string) {
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

// admin 控制器 登录验证
func AuthMiddleWare() gin.HandlerFunc {
	return func(c *gin.Context) {
		conf:=GetConf()
		user:=conf.WebUi.User
		password:=conf.WebUi.Password
		str1:=user+password
		h:= md5.New()
		h.Write([]byte(str1))
		md51:=hex.EncodeToString(h.Sum(nil))
		if cookie, err := c.Request.Cookie("userinfo"); err == nil {
			value := cookie.Value
			if value == md51 {
				c.Next()
				return
			}
		}
		c.HTML(http.StatusOK,"jump.html",gin.H{
			"url":"/index/login",
			"timeout":"3",
			"code":0,//1为success,0为error
			"msg":"请登录后再访问",
		})
		//c.Redirect(http.StatusMovedPermanently, "/index/login")
		c.Abort()
		return
	}
}