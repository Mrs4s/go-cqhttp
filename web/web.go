package web

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/Mrs4s/MiraiGo/binary"
	"github.com/Mrs4s/MiraiGo/client"
	"github.com/Mrs4s/go-cqhttp/coolq"
	"github.com/Mrs4s/go-cqhttp/global"
	"github.com/Mrs4s/go-cqhttp/server"
	"github.com/gin-gonic/gin"
	"github.com/gobuffalo/packr/v2"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	asciiart "github.com/yinghau76/go-ascii-art"
	"html/template"
	"image"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"
)

type webServer struct {
	engine  *gin.Engine
	bot     *coolq.CQBot
	Cli     *client.QQClient
	Conf    *global.JsonConfig
	Console *bufio.Reader
}

var WebServer = &webServer{}

func (s *webServer) Run(addr string, cli *client.QQClient) *coolq.CQBot {
	s.Cli = cli
	s.Conf = GetConf()
	gin.SetMode(gin.ReleaseMode)
	s.engine = gin.New()
	// 自动加载模板
	t := template.New("tmp")
	//func 函数映射 全局模板可用
	t.Funcs(template.FuncMap{
		"getYear":        GetYear,
		"formatAsDate":   FormatAsDate,
		"getConf":        GetConf,
		"getDate":        GetDate,
		"getavator":      Getavator,
		"getServerInfo":  GetServerInfo,
		"formatFileSize": FormatFileSize,
	})
	//从二进制中加载模板（后缀必须.html)
	t, _ = s.LoadTemplate(t)
	s.engine.SetHTMLTemplate(t)
	//静态资源
	assets := packr.New("assets", "../template/assets")
	//s.engine.Static("/assets", "./template/assets")
	s.engine.StaticFS("/assets", assets)
	s.engine.GET("/", func(c *gin.Context) {
		c.Redirect(302, "/index/login")
	})
	//通用路由
	s.engine.Any("/admin/:action", AuthMiddleWare(), s.admin)
	s.engine.Any("/index/:action", s.index)
	s.engine.Use(func(c *gin.Context) {
		if c.Request.Method != "GET" && c.Request.Method != "POST" {
			log.Warnf("已拒绝客户端 %v 的请求: 方法错误", c.Request.RemoteAddr)
			c.Status(404)
			c.Abort()
		}
		if c.Request.Method == "POST" && strings.Contains(c.Request.Header.Get("Content-Type"), "application/json") {
			d, err := c.GetRawData()
			if err != nil {
				log.Warnf("获取请求 %v 的Body时出现错误: %v", c.Request.RequestURI, err)
				c.Status(400)
				c.Abort()
			}
			if !gjson.ValidBytes(d) {
				log.Warnf("已拒绝客户端 %v 的请求: 非法Json", c.Request.RemoteAddr)
				c.Status(400)
				c.Abort()
			}
			c.Set("json_body", gjson.ParseBytes(d))
		}
		c.Next()
	})
	go func() {
		log.Infof("miraigo webui 服务器已启动: %v", addr)
		err := s.engine.Run(addr)
		if err != nil {
			log.Error(err)
			log.Infof("请检查端口是否被占用.")
			time.Sleep(time.Second * 5)
			os.Exit(1)
		}
	}()
	s.Dologin()
	s.UpServer()
	b := s.bot //外部引入 bot对象，用于操作bot
	return b
}

func (s *webServer) Dologin() {
	s.Console = bufio.NewReader(os.Stdin)
	conf := GetConf()
	cli := s.Cli
	rsp, err := cli.Login()
	for {
		global.Check(err)
		if !rsp.Success {
			switch rsp.Error {
			case client.NeedCaptcha:
				_ = ioutil.WriteFile("captcha.jpg", rsp.CaptchaImage, 0644)
				img, _, _ := image.Decode(bytes.NewReader(rsp.CaptchaImage))
				fmt.Println(asciiart.New("image", img).Art)
				log.Warn("请输入验证码 (captcha.jpg)： (http://127.0.0.1/admin/web_write 输入)")
				//text, _ := s.Console.ReadString('\n')
				var text string
				for {
					file := "input.txt"
					text = global.ReadAllText(file)
					if text != "" {
						global.DelFile(file)
						break
					}
				}
				rsp, err = cli.SubmitCaptcha(strings.ReplaceAll(text, "\n", ""), rsp.CaptchaSign)
				continue
			case client.UnsafeDeviceError:
				log.Warnf("账号已开启设备锁，请前往 -> %v <- 验证并重启Bot.", rsp.VerifyUrl)
				log.Infof(" (http://127.0.0.1/admin/web_write 确认后继续)....")
				//_, _ = s.Console.ReadString('\n')
				var text string
				for {
					file := "input.txt"
					text = global.ReadAllText(file)
					if text != "" {
						global.DelFile(file)
						break
					}
				}
				continue
			case client.OtherLoginError, client.UnknownLoginError:
				log.Fatalf("登录失败: %v", rsp.ErrorMessage)
				return
			}
		}
		break
	}
	log.Infof("登录成功 欢迎使用: %v", cli.Nickname)
	time.Sleep(time.Second)
	log.Info("开始加载好友列表...")
	global.Check(cli.ReloadFriendList())
	log.Infof("共加载 %v 个好友.", len(cli.FriendList))
	log.Infof("开始加载群列表...")
	global.Check(cli.ReloadGroupList())
	log.Infof("共加载 %v 个群.", len(cli.GroupList))
	s.bot = coolq.NewQQBot(cli, conf)
	if conf.PostMessageFormat != "string" && conf.PostMessageFormat != "array" {
		log.Warnf("post_message_format 配置错误, 将自动使用 string")
		coolq.SetMessageFormat("string")
		return
	} else {
		coolq.SetMessageFormat(conf.PostMessageFormat)
	}
	if conf.RateLimit.Enabled {
		global.InitLimiter(conf.RateLimit.Frequency, conf.RateLimit.BucketSize)
	}
	log.Info("正在加载事件过滤器.")
	global.BootFilter()
	coolq.IgnoreInvalidCQCode = conf.IgnoreInvalidCQCode
	coolq.ForceFragmented = conf.ForceFragmented
	log.Info("资源初始化完成, 开始处理信息.")
	log.Info("アトリは、高性能ですから!")
	cli.OnDisconnected(func(bot *client.QQClient, e *client.ClientDisconnectedEvent) {
		if conf.ReLogin.Enabled {
			var times uint = 1
			for {
				if conf.ReLogin.MaxReloginTimes == 0 {
				} else if times > conf.ReLogin.MaxReloginTimes {
					break
				}
				log.Warnf("Bot已离线 (%v)，将在 %v 秒后尝试重连. 重连次数：%v",
					e.Message, conf.ReLogin.ReLoginDelay, times)
				times++
				time.Sleep(time.Second * time.Duration(conf.ReLogin.ReLoginDelay))
				rsp, err := cli.Login()
				if err != nil {
					log.Errorf("重连失败: %v", err)
					continue
				}
				if !rsp.Success {
					switch rsp.Error {
					case client.NeedCaptcha:
						log.Fatalf("重连失败: 需要验证码. (验证码处理正在开发中)")
					case client.UnsafeDeviceError:
						log.Fatalf("重连失败: 设备锁")
					default:
						log.Errorf("重连失败: %v", rsp.ErrorMessage)
						continue
					}
				}
				log.Info("重连成功")
				return
			}
			log.Fatal("重连失败: 重连次数达到设置的上限值")
		}
		s.bot.Release()
		log.Fatalf("Bot已离线：%v", e.Message)
	})
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
	Root       string
	Version    string
	Hostname   string
	Interfaces interface{}
	Goarch     string
	Goos       string
	//VirtualMemory *mem.VirtualMemoryStat
	Sys         uint64
	CpuInfoStat struct {
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
		conf := GetConf()
		user := conf.WebUi.User
		password := conf.WebUi.Password
		str1 := user + password
		h := md5.New()
		h.Write([]byte(str1))
		md51 := hex.EncodeToString(h.Sum(nil))
		if cookie, err := c.Request.Cookie("userinfo"); err == nil {
			value := cookie.Value
			if value == md51 {
				c.Next()
				return
			}
		}
		c.HTML(http.StatusOK, "index/jump.html", gin.H{
			"url":     "/index/login",
			"timeout": "3",
			"code":    0, //1为success,0为error
			"msg":     "请登录后再访问",
		})
		//c.Redirect(http.StatusMovedPermanently, "/index/login")
		c.Abort()
		return
	}
}

// loadTemplate loads templates by packr 将html 打包到二进制包
func (s *webServer) LoadTemplate(t *template.Template) (*template.Template, error) {
	box := packr.New("tmp", "../template/html")
	for _, file := range box.List() {
		if !strings.HasSuffix(file, ".html") {
			continue
		}
		h, err := box.FindString(file)
		if err != nil {
			return nil, err
		}
		//拼接方式，组装模板  admin/index.html 这种，方便调用
		t, err = t.New(strings.Replace(file, "html/", "", 1)).Parse(h)
		if err != nil {
			return nil, err
		}
	}
	return t, nil
}

func (s *webServer) DoRelogin() {
	conf := GetConf()
	OldConf := s.Conf
	cli := client.NewClient(conf.Uin, conf.Password)
	cli.OnLog(func(c *client.QQClient, e *client.LogEvent) {
		switch e.Type {
		case "INFO":
			log.Info("Protocol -> " + e.Message)
		case "ERROR":
			log.Error("Protocol -> " + e.Message)
		case "DEBUG":
			log.Debug("Protocol -> " + e.Message)
		}
	})
	cli.OnServerUpdated(func(bot *client.QQClient, e *client.ServerUpdatedEvent) {
		log.Infof("收到服务器地址更新通知, 将在下一次重连时应用. 服务器地址: %v:%v 服务器位置: %v", e.Servers[0].Server, e.Servers[0].Port, e.Servers[0].Location)
		_ = ioutil.WriteFile("servers.bin", binary.NewWriterF(func(w *binary.Writer) {
			w.WriteUInt16(uint16(len(e.Servers)))
			for _, s := range e.Servers {
				if !strings.Contains(s.Server, "com") {
					w.WriteString(s.Server)
					w.WriteUInt16(uint16(s.Port))
				}
			}
		}), 0644)
	})
	if global.PathExists("servers.bin") {
		if data, err := ioutil.ReadFile("servers.bin"); err == nil {
			r := binary.NewReader(data)
			r.ReadUInt16()
			cli.CustomServer = &net.TCPAddr{
				IP:   net.ParseIP(r.ReadString()),
				Port: int(r.ReadUInt16()),
			}
		}
	}
	s.Cli = cli
	s.Dologin()
	//关闭之前的 server
	if OldConf.HttpConfig != nil && OldConf.HttpConfig.Enabled {
		server.HttpServer.ShutDown()
	}
	if OldConf.WSConfig != nil && OldConf.WSConfig.Enabled {
		//server.WebsocketServer.ShutDown()
	}
	//s.UpServer()
	s.ReloadServer()
}

func (s *webServer) UpServer() {
	conf := GetConf()
	if conf.HttpConfig != nil && conf.HttpConfig.Enabled {
		go server.HttpServer.Run(fmt.Sprintf("%s:%d", conf.HttpConfig.Host, conf.HttpConfig.Port), conf.AccessToken, s.bot)
		for k, v := range conf.HttpConfig.PostUrls {
			server.NewHttpClient().Run(k, v, conf.HttpConfig.Timeout, s.bot)
		}
	}
	if conf.WSConfig != nil && conf.WSConfig.Enabled {
		go server.WebsocketServer.Run(fmt.Sprintf("%s:%d", conf.WSConfig.Host, conf.WSConfig.Port), conf.AccessToken, s.bot)
	}
	for _, rc := range conf.ReverseServers {
		go server.NewWebsocketClient(rc, conf.AccessToken, s.bot).Run()
	}
}

// 暂不支持ws服务的重启
func (s *webServer) ReloadServer() {
	conf := GetConf()
	if conf.HttpConfig != nil && conf.HttpConfig.Enabled {
		go server.HttpServer.Run(fmt.Sprintf("%s:%d", conf.HttpConfig.Host, conf.HttpConfig.Port), conf.AccessToken, s.bot)
		for k, v := range conf.HttpConfig.PostUrls {
			server.NewHttpClient().Run(k, v, conf.HttpConfig.Timeout, s.bot)
		}
	}
	for _, rc := range conf.ReverseServers {
		go server.NewWebsocketClient(rc, conf.AccessToken, s.bot).Run()
	}
}
