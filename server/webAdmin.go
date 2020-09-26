package server

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/Mrs4s/MiraiGo/client"
	"github.com/Mrs4s/go-cqhttp/coolq"
	"github.com/Mrs4s/go-cqhttp/global"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"github.com/yinghau76/go-ascii-art"
	"image"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

var WebInput = make(chan string, 1) //长度1，用于阻塞

var Console = make(chan os.Signal, 1)

type webServer struct {
	engine  *gin.Engine
	bot     *coolq.CQBot
	Cli     *client.QQClient
	Conf    *global.JsonConfig
	Console *bufio.Reader
}

var WebServer = &webServer{}

// admin 子站的 路由映射
var HttpuriAdmin = map[string]func(s *webServer, c *gin.Context){
	"do_restart":        AdminDoRestart,       //热重启
	"get_web_write":     AdminWebWrite,        //获取是否验证码输入
	"do_web_write":      AdminDoWebWrite,      //web上进行输入操作
	"do_restart_docker": AdminDoRestartDocker, //直接停止（依赖supervisord/docker）重新拉起
}


func (s *webServer) Run(addr string, cli *client.QQClient) *coolq.CQBot {
	s.Cli = cli
	s.Conf = GetConf()
	gin.SetMode(gin.ReleaseMode)
	s.engine = gin.New()

	//通用路由
	s.engine.Any("/admin/:action", AuthMiddleWare(), s.admin)

	go func() {
		log.Infof("miraigo adminapi 服务器已启动: %v", addr)
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
		var text string
		if !rsp.Success {
			switch rsp.Error {
			case client.NeedCaptcha:
				_ = ioutil.WriteFile("captcha.jpg", rsp.CaptchaImage, 0644)
				img, _, _ := image.Decode(bytes.NewReader(rsp.CaptchaImage))
				fmt.Println(asciiart.New("image", img).Art)
				if conf.WebUi.WebInput {
					log.Warn("请输入验证码 (captcha.jpg)： (http://127.0.0.1/admin/web_write 输入)")
					text = <-WebInput
				} else {
					log.Warn("请输入验证码 (captcha.jpg)： (Enter 提交)")
					text, _ = s.Console.ReadString('\n')
				}
				rsp, err = cli.SubmitCaptcha(strings.ReplaceAll(text, "\n", ""), rsp.CaptchaSign)
				global.DelFile("captcha.jpg")
				continue
			case client.UnsafeDeviceError:
				log.Warnf("账号已开启设备锁，请前往 -> %v <- 验证并重启Bot.", rsp.VerifyUrl)
				if conf.WebUi.WebInput {
					log.Infof(" (http://127.0.0.1/admin/web_write 确认后继续)....")
					text = <-WebInput
				} else {
					log.Infof(" 按 Enter 继续....")
					_, _ = s.Console.ReadString('\n')
				}
				log.Info(text)
				return
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

// 获取当前配置文件信息
func GetConf() *global.JsonConfig {
	conf := global.Load("config.json")
	return conf
}

// admin 控制器 登录验证
func AuthMiddleWare() gin.HandlerFunc {
	return func(c *gin.Context) {
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
		conf := GetConf()
		authToken := conf.AccessToken
		if auth := c.Request.Header.Get("Authorization"); auth != "" {
			if strings.SplitN(auth, " ", 2)[1] != authToken {
				c.AbortWithStatus(401)
				return
			}
		} else if c.Query("access_token") != authToken {
			c.AbortWithStatus(401)
			return
		} else {
			c.Next()
		}
	}
}

func (s *webServer) DoRelogin() {
	conf := GetConf()
	OldConf := s.Conf
	cli := client.NewClient(conf.Uin, conf.Password)
	log.Info("开始尝试登录并同步消息...")
	log.Infof("使用协议: %v", func() string {
		switch client.SystemDeviceInfo.Protocol {
		case client.AndroidPad:
			return "Android Pad"
		case client.AndroidPhone:
			return "Android Phone"
		case client.AndroidWatch:
			return "Android Watch"
		}
		return "未知"
	}())
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
		log.Infof("收到服务器地址更新通知, 将在下一次重连时应用. ")
	})
	s.Cli = cli
	s.Dologin()
	//关闭之前的 server
	if OldConf.HttpConfig != nil && OldConf.HttpConfig.Enabled {
		HttpServer.ShutDown()
	}
	//if OldConf.WSConfig != nil && OldConf.WSConfig.Enabled {
	//	server.WsShutdown()
	//}
	//s.UpServer()
	s.ReloadServer()
}

func (s *webServer) UpServer() {
	conf := GetConf()
	if conf.HttpConfig != nil && conf.HttpConfig.Enabled {
		go HttpServer.Run(fmt.Sprintf("%s:%d", conf.HttpConfig.Host, conf.HttpConfig.Port), conf.AccessToken, s.bot)
		for k, v := range conf.HttpConfig.PostUrls {
			NewHttpClient().Run(k, v, conf.HttpConfig.Timeout, s.bot)
		}
	}
	if conf.WSConfig != nil && conf.WSConfig.Enabled {
		go WebsocketServer.Run(fmt.Sprintf("%s:%d", conf.WSConfig.Host, conf.WSConfig.Port), conf.AccessToken, s.bot)
	}
	for _, rc := range conf.ReverseServers {
		go NewWebsocketClient(rc, conf.AccessToken, s.bot).Run()
	}
}

// 暂不支持ws服务的重启
func (s *webServer) ReloadServer() {
	conf := GetConf()
	if conf.HttpConfig != nil && conf.HttpConfig.Enabled {
		go HttpServer.Run(fmt.Sprintf("%s:%d", conf.HttpConfig.Host, conf.HttpConfig.Port), conf.AccessToken, s.bot)
		for k, v := range conf.HttpConfig.PostUrls {
			NewHttpClient().Run(k, v, conf.HttpConfig.Timeout, s.bot)
		}
	}
	for _, rc := range conf.ReverseServers {
		go NewWebsocketClient(rc, conf.AccessToken, s.bot).Run()
	}
}


// 热重启
func AdminDoRestart(s *webServer, c *gin.Context) {
	s.DoRelogin()
	c.JSON(200, coolq.OK(coolq.MSG{}))
	return
}

// 冷重启
func AdminDoRestartDocker(s *webServer, c *gin.Context) {
	Console <- os.Kill
	c.JSON(200, coolq.OK(coolq.MSG{}))
	return
}

// web输入 html 页面
func AdminWebWrite(s *webServer, c *gin.Context) {
	pic := global.ReadAllText("captcha.jpg")
	var picbase64 string
	if pic != "" {
		input := []byte(pic)
		// base64编码
		picbase64 = base64.StdEncoding.EncodeToString(input)
	}
	c.JSON(200, coolq.OK(coolq.MSG{
		"pic":       pic,
		"picbase64": picbase64,
	}))
}

// web输入 处理
func AdminDoWebWrite(s *webServer, c *gin.Context) {
	input := c.PostForm("input")
	WebInput <- input
	//global.WriteAllText("input.txt", input)
	c.JSON(200, coolq.OK(coolq.MSG{}))
}
