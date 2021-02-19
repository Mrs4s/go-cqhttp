package server

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Mrs4s/MiraiGo/utils"
	"github.com/gin-contrib/pprof"

	"github.com/Mrs4s/MiraiGo/client"
	"github.com/Mrs4s/go-cqhttp/coolq"
	"github.com/Mrs4s/go-cqhttp/global"
	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	asciiart "github.com/yinghau76/go-ascii-art"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

// WebInput 网页输入channel
var WebInput = make(chan string, 1) //长度1，用于阻塞

// Console 控制台channel
var Console = make(chan os.Signal, 1)

// Restart 重启信号监听channel
var Restart = make(chan struct{}, 1)

// JSONConfig go-cqhttp配置
var JSONConfig *global.JSONConfig

type webServer struct {
	engine  *gin.Engine
	bot     *coolq.CQBot
	Cli     *client.QQClient
	Conf    *global.JSONConfig //old config
	Console *bufio.Reader
}

// WebServer Admin子站的Server
var WebServer = &webServer{}

// APIAdminRoutingTable Admin子站的路由映射
var APIAdminRoutingTable = map[string]func(s *webServer, c *gin.Context){
	"do_restart":         AdminDoRestart,         //热重启
	"do_process_restart": AdminProcessRestart,    //进程重启
	"get_web_write":      AdminWebWrite,          //获取是否验证码输入
	"do_web_write":       AdminDoWebWrite,        //web上进行输入操作
	"do_restart_docker":  AdminDoRestartDocker,   //直接停止（依赖supervisord/docker）重新拉起
	"do_config_base":     AdminDoConfigBase,      //修改config.json中的基础部分
	"do_config_http":     AdminDoConfigHTTP,      //修改config.json的http部分
	"do_config_ws":       AdminDoConfigWS,        //修改config.json的正向ws部分
	"do_config_reverse":  AdminDoConfigReverseWS, //修改config.json 中的反向ws部分
	"do_config_json":     AdminDoConfigJSON,      //直接修改 config.json配置
	"get_config_json":    AdminGetConfigJSON,     //拉取 当前的config.json配置
}

// Failed 构建失败返回MSG
func Failed(code int, msg string) coolq.MSG {
	return coolq.MSG{"data": nil, "retcode": code, "status": "failed", "msg": msg}
}

func (s *webServer) Run(addr string, cli *client.QQClient) *coolq.CQBot {
	s.Cli = cli
	s.Conf = GetConf()
	JSONConfig = s.Conf
	gin.SetMode(gin.ReleaseMode)
	s.engine = gin.New()

	s.engine.Use(AuthMiddleWare())

	// 通用路由
	s.engine.Any("/admin/:action", s.admin)

	go func() {
		// 开启端口监听
		if s.Conf.WebUI != nil && s.Conf.WebUI.Enabled {
			if Debug {
				pprof.Register(s.engine)
				log.Debugf("pprof 性能分析服务已启动在 http://%v/debug/pprof, 如果有任何性能问题请下载报告并提交给开发者", addr)
				time.Sleep(time.Second * 3)
			}
			log.Infof("Admin API 服务器已启动: %v", addr)
			err := s.engine.Run(addr)
			if err != nil {
				log.Error(err)
				log.Infof("请检查端口是否被占用.")
				c := make(chan os.Signal, 1)
				signal.Notify(c, os.Interrupt, syscall.SIGTERM)
				<-c
				os.Exit(1)
			}
		} else {
			// 关闭端口监听
			c := make(chan os.Signal, 1)
			signal.Notify(c, os.Interrupt, syscall.SIGTERM)
			<-c
			os.Exit(1)
		}
	}()
	s.Dologin()
	s.UpServer()
	b := s.bot // 外部引入 bot对象，用于操作bot
	return b
}

// logincore 登录核心实现
func (s *webServer) logincore(relogin bool) {

	s.Console = bufio.NewReader(os.Stdin)
	readLine := func() (str string) {
		str, _ = s.Console.ReadString('\n')
		str = strings.TrimSpace(str)
		return
	}

	if s.Cli.Online {
		log.Warn("Bot已登录")
		return
	}

	var times uint = 1 // 重试次数
	for res, err := s.Cli.Login(); ; res, err = s.Cli.Login() {

		var text string
		count := 0

		if res == nil {
			goto Relogin
		}

	Again: // 不执行 s.Cli.Login() 的循环，适用输入验证码等更新 res 的操作
		if err == nil && res.Success { // 登录成功
			break
		} else if err == client.ErrAlreadyOnline {
			break
		}

		switch res.Error {
		case client.SliderNeededError:
			log.Warnf("登录需要滑条验证码, 请选择解决方案: ")
			log.Warnf("1. 自行抓包. (推荐)")
			log.Warnf("2. 使用Cef自动处理.")
			log.Warnf("3. 不提交滑块并继续.(可能会导致上网环境异常错误)")
			log.Warnf("详细信息请参考文档 -> https://github.com/Mrs4s/go-cqhttp/blob/master/docs/slider.md <-")
			if s.Conf.WebUI != nil && s.Conf.WebUI.WebInput {
				log.Warnf("请输入(1 - 3)： (http://%s:%d/admin/do_web_write 输入)", s.Conf.WebUI.Host, s.Conf.WebUI.WebUIPort)
				text = <-WebInput
			} else {
				log.Warn("请输入(1 - 3)：")
				text = readLine()
			}
			if strings.Contains(text, "1") {
				log.Warnf("请用浏览器打开 -> %v <- 并获取Ticket.", res.VerifyUrl)
				if s.Conf.WebUI != nil && s.Conf.WebUI.WebInput {
					log.Warnf("请输入Ticket： (http://%s:%d/admin/do_web_write 输入)", s.Conf.WebUI.Host, s.Conf.WebUI.WebUIPort)
					text = <-WebInput
				} else {
					log.Warn("请输入Ticket： (Enter 提交)")
					text = readLine()
				}
				res, err = s.Cli.SubmitTicket(strings.TrimSpace(text))
				goto Again
			}
			if strings.Contains(text, "3") {
				s.Cli.AllowSlider = false
				s.Cli.Disconnect()
				continue
			}
			id := utils.RandomStringRange(6, "0123456789")
			log.Warnf("滑块ID为 %v 请在30S内处理.", id)
			ticket, err := global.GetSliderTicket(res.VerifyUrl, id)
			if err != nil {
				log.Warnf("错误: " + err.Error())
				os.Exit(0)
			}
			res, err = s.Cli.SubmitTicket(ticket)
			if err != nil {
				log.Warnf("错误: " + err.Error())
				continue // 尝试重新登录
			}
			goto Again
		case client.NeedCaptcha:
			_ = ioutil.WriteFile("captcha.jpg", res.CaptchaImage, 0644)
			img, _, _ := image.Decode(bytes.NewReader(res.CaptchaImage))
			fmt.Println(asciiart.New("image", img).Art)
			if s.Conf.WebUI != nil && s.Conf.WebUI.WebInput {
				log.Warnf("请输入验证码 (captcha.jpg)： (http://%s:%d/admin/do_web_write 输入)", s.Conf.WebUI.Host, s.Conf.WebUI.WebUIPort)
				text = <-WebInput
			} else {
				log.Warn("请输入验证码 (captcha.jpg)： (Enter 提交)")
				text = readLine()
			}
			global.DelFile("captcha.jpg")
			res, err = s.Cli.SubmitCaptcha(strings.ReplaceAll(text, "\n", ""), res.CaptchaSign)
			goto Again
		case client.SMSNeededError:
			if s.Conf.WebUI != nil && s.Conf.WebUI.WebInput {
				log.Warnf("账号已开启设备锁, 已向手机 %v 发送短信验证码.", res.SMSPhone)
			} else {
				log.Warnf("账号已开启设备锁, 按下 Enter 向手机 %v 发送短信验证码.", res.SMSPhone)
				readLine()
			}
			if !s.Cli.RequestSMS() {
				log.Warnf("发送验证码失败，可能是请求过于频繁.")
				time.Sleep(time.Second * 5)
				continue
			}
			if s.Conf.WebUI != nil && s.Conf.WebUI.WebInput {
				log.Warnf("请输入短信验证码： (http://%s:%d/admin/do_web_write 输入)", s.Conf.WebUI.Host, s.Conf.WebUI.WebUIPort)
				text = <-WebInput
			} else {
				log.Warn("请输入短信验证码： (Enter 提交)")
				text = readLine()
			}
			res, err = s.Cli.SubmitSMS(strings.ReplaceAll(strings.ReplaceAll(text, "\n", ""), "\r", ""))
			goto Again
		case client.SMSOrVerifyNeededError:
			log.Warnf("账号已开启设备锁，请选择验证方式:")
			log.Warnf("1. 向手机 %v 发送短信验证码", res.SMSPhone)
			log.Warnf("2. 使用手机QQ扫码验证.")
			if s.Conf.WebUI != nil && s.Conf.WebUI.WebInput {
				log.Warnf("请输入(1 - 2)： (http://%s:%d/admin/do_web_write 输入)", s.Conf.WebUI.Host, s.Conf.WebUI.WebUIPort)
				text = <-WebInput
			} else {
				log.Warn("请输入(1 - 2)：")
				text = readLine()
			}
			if strings.Contains(text, "1") {
				if !s.Cli.RequestSMS() {
					log.Warnf("发送验证码失败，可能是请求过于频繁.")
					time.Sleep(time.Second * 5)
					os.Exit(0)
				}
				if s.Conf.WebUI != nil && s.Conf.WebUI.WebInput {
					log.Warnf("请输入短信验证码：  (http://%s:%d/admin/do_web_write 输入)....", s.Conf.WebUI.Host, s.Conf.WebUI.WebUIPort)
					text = <-WebInput
				} else {
					log.Warn("请输入短信验证码： (Enter 提交)")
					text = readLine()
				}
				res, err = s.Cli.SubmitSMS(strings.ReplaceAll(strings.ReplaceAll(text, "\n", ""), "\r", ""))
				goto Again
			}
			log.Warnf("请前往 -> %v <- 验证.", res.VerifyUrl)
			log.Infof("按 Enter 继续....")
			readLine()
			continue
		case client.UnsafeDeviceError:
			log.Warnf("账号已开启设备锁，请前往 -> %v <- 验证.", res.VerifyUrl)
			if s.Conf.WebUI != nil && s.Conf.WebUI.WebInput {
				log.Infof(" (http://%s:%d/admin/do_web_write 确认后继续)....", s.Conf.WebUI.Host, s.Conf.WebUI.WebUIPort)
				text = <-WebInput
			} else {
				log.Infof("按 Enter 继续....")
				readLine()
			}
			log.Info(text)
			continue
		case client.OtherLoginError, client.UnknownLoginError:
			msg := res.ErrorMessage
			if strings.Contains(msg, "版本") {
				msg = "密码错误或账号被冻结"
			}
			if strings.Contains(msg, "上网环境") && count < 5 {
				s.Cli.Disconnect()
				log.Warnf("错误: 当前上网环境异常. 将更换服务器并重试.")
				count++
				time.Sleep(time.Second)
				continue
			}
			if strings.Contains(msg, "冻结") {
				log.Fatalf("账号被冻结, 放弃重连")
			}
			log.Warnf("登录失败: %v", msg)
			log.Infof("按 Enter 继续....")
			readLine()
			os.Exit(0)
		}

	Relogin:
		if relogin {
			if times > s.Conf.ReLogin.MaxReloginTimes && s.Conf.ReLogin.MaxReloginTimes != 0 {
				log.Fatal("重连失败: 重连次数达到设置的上限值")
				s.bot.Release()
				return
			}
			log.Warnf("将在 %v 秒后尝试重连. 重连次数：%v", s.Conf.ReLogin.ReLoginDelay, times)
			times++
			time.Sleep(time.Second * time.Duration(s.Conf.ReLogin.ReLoginDelay))
			s.Cli.Disconnect()
			continue
		}
	}
	if relogin {
		log.Info("重连成功")
	}
}

// Dologin 主程序登录
func (s *webServer) Dologin() {

	s.Cli.AllowSlider = true
	s.logincore(false)
	log.Infof("登录成功 欢迎使用: %v", s.Cli.Nickname)
	log.Info("开始加载好友列表...")
	global.Check(s.Cli.ReloadFriendList())
	log.Infof("共加载 %v 个好友.", len(s.Cli.FriendList))
	log.Infof("开始加载群列表...")
	global.Check(s.Cli.ReloadGroupList())
	log.Infof("共加载 %v 个群.", len(s.Cli.GroupList))
	s.bot = coolq.NewQQBot(s.Cli, s.Conf)
	if s.Conf.PostMessageFormat != "string" && s.Conf.PostMessageFormat != "array" {
		log.Warnf("post_message_format 配置错误, 将自动使用 string")
		coolq.SetMessageFormat("string")
	} else {
		coolq.SetMessageFormat(s.Conf.PostMessageFormat)
	}
	if s.Conf.RateLimit.Enabled {
		global.InitLimiter(s.Conf.RateLimit.Frequency, s.Conf.RateLimit.BucketSize)
	}
	log.Info("正在加载事件过滤器.")
	global.BootFilter()
	coolq.IgnoreInvalidCQCode = s.Conf.IgnoreInvalidCQCode
	coolq.SplitURL = s.Conf.FixURL
	coolq.ForceFragmented = s.Conf.ForceFragmented
	log.Info("资源初始化完成, 开始处理信息.")
	log.Info("アトリは、高性能ですから!")

	s.Cli.OnDisconnected(func(q *client.QQClient, e *client.ClientDisconnectedEvent) {
		if !s.Conf.ReLogin.Enabled {
			return
		}
		log.Warnf("Bot已离线 (%v)，尝试重连", e.Message)
		s.logincore(true)
	})
}

func (s *webServer) admin(c *gin.Context) {
	action := c.Param("action")
	log.Debugf("WebServer接收到cgi调用: %v", action)
	if f, ok := APIAdminRoutingTable[action]; ok {
		f(s, c)
	} else {
		c.JSON(200, coolq.Failed(404))
	}
}

// GetConf 获取当前配置文件信息
func GetConf() *global.JSONConfig {
	if JSONConfig != nil {
		return JSONConfig
	}
	conf := global.LoadConfig("config.hjson")
	return conf
}

// AuthMiddleWare Admin控制器登录验证
func AuthMiddleWare() gin.HandlerFunc {
	return func(c *gin.Context) {
		conf := GetConf()
		// 处理跨域问题
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Headers", "Content-Type,AccessToken,X-CSRF-Token, Authorization, Token")
		c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, PATCH, DELETE")
		c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Content-Type")
		c.Header("Access-Control-Allow-Credentials", "true")
		// 放行所有OPTIONS方法，因为有的模板是要请求两次的
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
		}
		if strings.Contains(c.Request.URL.Path, "debug") {
			c.Next()
			return
		}
		// 处理请求
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

func (s *webServer) DoReLogin() { // TODO: 协议层的 ReLogin
	JSONConfig = nil
	conf := GetConf()
	OldConf := s.Conf
	cli := client.NewClient(conf.Uin, conf.Password)
	log.Info("开始尝试登录并同步消息...")
	log.Infof("使用协议: %v", func() string {
		switch client.SystemDeviceInfo.Protocol {
		case client.IPad:
			return "iPad"
		case client.AndroidPhone:
			return "Android Phone"
		case client.AndroidWatch:
			return "Android Watch"
		case client.MacOS:
			return "MacOS"
		}
		return "未知"
	}())
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
	cli.OnServerUpdated(func(bot *client.QQClient, e *client.ServerUpdatedEvent) bool {
		if !conf.UseSSOAddress {
			log.Infof("收到服务器地址更新通知, 根据配置文件已忽略.")
			return false
		}
		log.Infof("收到服务器地址更新通知, 将在下一次重连时应用. ")
		return true
	})
	s.Cli = cli
	s.Dologin()
	// 关闭之前的 server
	if OldConf.HTTPConfig != nil && OldConf.HTTPConfig.Enabled {
		cqHTTPServer.ShutDown()
	}
	// if OldConf.WSConfig != nil && OldConf.WSConfig.Enabled {
	// 	server.WsShutdown()
	// }
	// s.UpServer()

	s.ReloadServer()
	s.Conf = conf
}

func (s *webServer) UpServer() {
	conf := GetConf()
	if conf.HTTPConfig != nil && conf.HTTPConfig.Enabled {
		go cqHTTPServer.Run(fmt.Sprintf("%s:%d", conf.HTTPConfig.Host, conf.HTTPConfig.Port), conf.AccessToken, s.bot)
		for k, v := range conf.HTTPConfig.PostUrls {
			newHTTPClient().Run(k, v, conf.HTTPConfig.Timeout, s.bot)
		}
	}
	if conf.WSConfig != nil && conf.WSConfig.Enabled {
		go WebSocketServer.Run(fmt.Sprintf("%s:%d", conf.WSConfig.Host, conf.WSConfig.Port), conf.AccessToken, s.bot)
	}
	for _, rc := range conf.ReverseServers {
		go NewWebSocketClient(rc, conf.AccessToken, s.bot).Run()
	}
}

// 暂不支持ws服务的重启
func (s *webServer) ReloadServer() {
	conf := GetConf()
	if conf.HTTPConfig != nil && conf.HTTPConfig.Enabled {
		go cqHTTPServer.Run(fmt.Sprintf("%s:%d", conf.HTTPConfig.Host, conf.HTTPConfig.Port), conf.AccessToken, s.bot)
		for k, v := range conf.HTTPConfig.PostUrls {
			newHTTPClient().Run(k, v, conf.HTTPConfig.Timeout, s.bot)
		}
	}
	for _, rc := range conf.ReverseServers {
		go NewWebSocketClient(rc, conf.AccessToken, s.bot).Run()
	}
}

// AdminDoRestart 热重启
func AdminDoRestart(s *webServer, c *gin.Context) {
	s.bot.Release()
	s.bot = nil
	s.Cli = nil
	s.DoReLogin()
	c.JSON(200, coolq.OK(coolq.MSG{}))
}

// AdminProcessRestart 进程重启
func AdminProcessRestart(s *webServer, c *gin.Context) {
	Restart <- struct{}{}
	c.JSON(200, coolq.OK(coolq.MSG{}))
}

// AdminDoRestartDocker 冷重启
func AdminDoRestartDocker(s *webServer, c *gin.Context) {
	Console <- os.Kill
	c.JSON(200, coolq.OK(coolq.MSG{}))
}

// AdminWebWrite web输入html页面
func AdminWebWrite(s *webServer, c *gin.Context) {
	pic := global.ReadAllText("captcha.jpg")
	var picbase64 string
	var ispic = false
	if pic != "" {
		input := []byte(pic)
		// base64编码
		picbase64 = base64.StdEncoding.EncodeToString(input)
		ispic = true
	}
	c.JSON(200, coolq.OK(coolq.MSG{
		"ispic":     ispic,     //为空则为 设备锁 或者没有需要输入
		"picbase64": picbase64, //web上显示图片
	}))
}

// AdminDoWebWrite web输入处理
func AdminDoWebWrite(s *webServer, c *gin.Context) {
	input := c.PostForm("input")
	WebInput <- input
	c.JSON(200, coolq.OK(coolq.MSG{}))
}

// AdminDoConfigBase 普通配置修改
func AdminDoConfigBase(s *webServer, c *gin.Context) {
	conf := GetConf()
	conf.Uin, _ = strconv.ParseInt(c.PostForm("uin"), 10, 64)
	conf.Password = c.PostForm("password")
	if c.PostForm("enable_db") == "true" {
		conf.EnableDB = true
	} else {
		conf.EnableDB = false
	}
	conf.AccessToken = c.PostForm("access_token")
	if err := conf.Save("config.hjson"); err != nil {
		log.Fatalf("保存 config.hjson 时出现错误: %v", err)
		c.JSON(200, Failed(502, "保存 config.hjson 时出现错误:"+fmt.Sprintf("%v", err)))
	} else {
		JSONConfig = nil
		c.JSON(200, coolq.OK(coolq.MSG{}))
	}
}

// AdminDoConfigHTTP HTTP配置修改
func AdminDoConfigHTTP(s *webServer, c *gin.Context) {
	conf := GetConf()
	p, _ := strconv.ParseUint(c.PostForm("port"), 10, 16)
	conf.HTTPConfig.Port = uint16(p)
	conf.HTTPConfig.Host = c.PostForm("host")
	if c.PostForm("enable") == "true" {
		conf.HTTPConfig.Enabled = true
	} else {
		conf.HTTPConfig.Enabled = false
	}
	t, _ := strconv.ParseInt(c.PostForm("timeout"), 10, 32)
	conf.HTTPConfig.Timeout = int32(t)
	if c.PostForm("post_url") != "" {
		conf.HTTPConfig.PostUrls[c.PostForm("post_url")] = c.PostForm("post_secret")
	}
	if err := conf.Save("config.hjson"); err != nil {
		log.Fatalf("保存 config.hjson 时出现错误: %v", err)
		c.JSON(200, Failed(502, "保存 config.hjson 时出现错误:"+fmt.Sprintf("%v", err)))
	} else {
		JSONConfig = nil
		c.JSON(200, coolq.OK(coolq.MSG{}))
	}
}

// AdminDoConfigWS ws配置修改
func AdminDoConfigWS(s *webServer, c *gin.Context) {
	conf := GetConf()
	p, _ := strconv.ParseUint(c.PostForm("port"), 10, 16)
	conf.WSConfig.Port = uint16(p)
	conf.WSConfig.Host = c.PostForm("host")
	if c.PostForm("enable") == "true" {
		conf.WSConfig.Enabled = true
	} else {
		conf.WSConfig.Enabled = false
	}
	if err := conf.Save("config.hjson"); err != nil {
		log.Fatalf("保存 config.hjson 时出现错误: %v", err)
		c.JSON(200, Failed(502, "保存 config.hjson 时出现错误:"+fmt.Sprintf("%v", err)))
	} else {
		JSONConfig = nil
		c.JSON(200, coolq.OK(coolq.MSG{}))
	}
}

// AdminDoConfigReverseWS 反向ws配置修改
func AdminDoConfigReverseWS(s *webServer, c *gin.Context) {
	conf := GetConf()
	conf.ReverseServers[0].ReverseAPIURL = c.PostForm("reverse_api_url")
	conf.ReverseServers[0].ReverseURL = c.PostForm("reverse_url")
	conf.ReverseServers[0].ReverseEventURL = c.PostForm("reverse_event_url")
	t, _ := strconv.ParseUint(c.PostForm("reverse_reconnect_interval"), 10, 16)
	conf.ReverseServers[0].ReverseReconnectInterval = uint16(t)
	if c.PostForm("enable") == "true" {
		conf.ReverseServers[0].Enabled = true
	} else {
		conf.ReverseServers[0].Enabled = false
	}
	if err := conf.Save("config.hjson"); err != nil {
		log.Fatalf("保存 config.hjson 时出现错误: %v", err)
		c.JSON(200, Failed(502, "保存 config.hjson 时出现错误:"+fmt.Sprintf("%v", err)))
	} else {
		JSONConfig = nil
		c.JSON(200, coolq.OK(coolq.MSG{}))
	}
}

// AdminDoConfigJSON config.hjson配置修改
func AdminDoConfigJSON(s *webServer, c *gin.Context) {
	conf := GetConf()
	JSON := c.PostForm("json")
	err := json.Unmarshal([]byte(JSON), &conf)
	if err != nil {
		log.Warnf("尝试加载配置文件 %v 时出现错误: %v", "config.hjson", err)
		c.JSON(200, Failed(502, "保存 config.hjson 时出现错误:"+fmt.Sprintf("%v", err)))
		return
	}
	if err := conf.Save("config.hjson"); err != nil {
		log.Fatalf("保存 config.hjson 时出现错误: %v", err)
		c.JSON(200, Failed(502, "保存 config.hjson 时出现错误:"+fmt.Sprintf("%v", err)))
	} else {
		JSONConfig = nil
		c.JSON(200, coolq.OK(coolq.MSG{}))
	}
}

// AdminGetConfigJSON 拉取config.hjson配置
func AdminGetConfigJSON(s *webServer, c *gin.Context) {
	conf := GetConf()
	c.JSON(200, coolq.OK(coolq.MSG{"config": conf}))

}
