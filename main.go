package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/Mrs4s/MiraiGo/client"
	"github.com/Mrs4s/go-cqhttp/coolq"
	"github.com/Mrs4s/go-cqhttp/global"
	"github.com/Mrs4s/go-cqhttp/server"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	log "github.com/sirupsen/logrus"
	easy "github.com/t-tomalak/logrus-easy-formatter"
	asciiart "github.com/yinghau76/go-ascii-art"
	"image"
	"io"
	"io/ioutil"
	"os"
	"os/signal"
	"path"
	"strconv"
	"strings"
	"time"
)

func init() {
	log.SetFormatter(&easy.Formatter{
		TimestampFormat: "2006-01-02 15:04:05",
		LogFormat:       "[%time%] [%lvl%]: %msg% \n",
	})
	w, err := rotatelogs.New(path.Join("logs", "%Y-%m-%d.log"), rotatelogs.WithRotationTime(time.Hour*24))
	if err == nil {
		log.SetOutput(io.MultiWriter(os.Stderr, w))
	}
	if !global.PathExists("data") {
		if err := os.Mkdir("data", 0777); err != nil {
			log.Fatalf("创建数据文件夹失败: %v", err)
		}
		if err := os.Mkdir(path.Join("data", "images"), 0777); err != nil {
			log.Fatalf("创建图片缓存文件夹失败: %v", err)
		}
	}
	if global.PathExists("cqhttp.json") {
		log.Info("发现 cqhttp.json 将在五秒后尝试导入配置，按 Ctrl+C 取消.")
		log.Warn("警告: 该操作会删除 cqhttp.json 并覆盖 config.json 文件.")
		time.Sleep(time.Second * 5)
		conf := global.CQHttpApiConfig{}
		if err := json.Unmarshal([]byte(global.ReadAllText("cqhttp.json")), &conf); err != nil {
			log.Fatalf("读取文件 cqhttp.json 失败: %v", err)
		}
		goConf := global.DefaultConfig()
		goConf.AccessToken = conf.AccessToken
		goConf.HttpConfig.Host = conf.Host
		goConf.HttpConfig.Port = conf.Port
		goConf.WSConfig.Host = conf.WSHost
		goConf.WSConfig.Port = conf.WSPort
		if conf.PostUrl != "" {
			goConf.HttpConfig.PostUrls[conf.PostUrl] = conf.Secret
		}
		if conf.UseWsReverse {
			goConf.ReverseServers[0].Enabled = true
			goConf.ReverseServers[0].ReverseUrl = conf.WSReverseUrl
			goConf.ReverseServers[0].ReverseApiUrl = conf.WSReverseApiUrl
			goConf.ReverseServers[0].ReverseEventUrl = conf.WSReverseEventUrl
			goConf.ReverseServers[0].ReverseReconnectInterval = conf.WSReverseReconnectInterval
		}
		if err := goConf.Save("config.json"); err != nil {
			log.Fatalf("保存 config.json 时出现错误: %v", err)
		}
		_ = os.Remove("cqhttp.json")
	}
}

func main() {
	console := bufio.NewReader(os.Stdin)
	var conf *global.JsonConfig
	if global.PathExists("config.json") || os.Getenv("UIN") == "" {
		conf = global.Load("config.json")
	} else if os.Getenv("UIN") != "" {
		log.Infof("将从环境变量加载配置.")
		uin, _ := strconv.ParseInt(os.Getenv("UIN"), 10, 64)
		pwd := os.Getenv("PASS")
		post := os.Getenv("HTTP_POST")
		conf = &global.JsonConfig{
			Uin:      uin,
			Password: pwd,
			HttpConfig: &global.GoCQHttpConfig{
				Enabled:  true,
				Host:     "0.0.0.0",
				Port:     5700,
				PostUrls: map[string]string{},
			},
			WSConfig: &global.GoCQWebsocketConfig{
				Enabled: true,
				Host:    "0.0.0.0",
				Port:    6700,
			},
			Debug: os.Getenv("DEBUG") == "true",
		}
		if post != "" {
			conf.HttpConfig.PostUrls[post] = os.Getenv("HTTP_SECRET")
		}
	}
	if conf == nil {
		err := global.DefaultConfig().Save("config.json")
		if err != nil {
			log.Fatalf("创建默认配置文件时出现错误: %v", err)
			return
		}
		log.Infof("默认配置文件已生成, 请编辑 config.json 后重启程序.")
		time.Sleep(time.Second * 5)
		return
	}
	if conf.Uin == 0 || conf.Password == "" {
		log.Warnf("请修改 config.json 以添加账号密码.")
		time.Sleep(time.Second * 5)
		return
	}
	if conf.Debug {
		log.SetLevel(log.DebugLevel)
		log.Warnf("已开启Debug模式.")
	}
	if !global.PathExists("device.json") {
		log.Warn("虚拟设备信息不存在, 将自动生成随机设备.")
		client.GenRandomDevice()
		_ = ioutil.WriteFile("device.json", client.SystemDeviceInfo.ToJson(), 0777)
		log.Info("已生成设备信息并保存到 device.json 文件.")
	} else {
		log.Info("将使用 device.json 内的设备信息运行Bot.")
		if err := client.SystemDeviceInfo.ReadJson([]byte(global.ReadAllText("device.json"))); err != nil {
			log.Fatalf("加载设备信息失败: %v", err)
		}
	}
	log.Info("Bot将在5秒后登录并开始信息处理, 按 Ctrl+C 取消.")
	time.Sleep(time.Second * 5)
	log.Info("开始尝试登录并同步消息...")
	cli := client.NewClient(conf.Uin, conf.Password)
	rsp, err := cli.Login()
	for {
		global.Check(err)
		if !rsp.Success {
			switch rsp.Error {
			case client.NeedCaptcha:
				ioutil.WriteFile("captcha.jpg", rsp.CaptchaImage, 0644)
				img, _, _ := image.Decode(bytes.NewReader(rsp.CaptchaImage))
				fmt.Println(asciiart.New("image", img).Art)
				log.Warn("请输入验证码：(图片已保存至 captcha.jpg, 输入后 Enter 提交)")
				text, _ := console.ReadString('\n')
				rsp, err = cli.SubmitCaptcha(strings.ReplaceAll(text, "\n", ""), rsp.CaptchaSign)
				continue
			case client.UnsafeDeviceError:
				log.Warnf("账号已开启设备锁，请前往 -> %v <- 验证并重启Bot.", rsp.VerifyUrl)
				log.Infof(" 按 Enter 继续....")
				_, _ = console.ReadString('\n')
				return
			case client.OtherLoginError, client.UnknownLoginError:
				log.Fatalf("登录失败: %v", rsp.ErrorMessage)
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
	b := coolq.NewQQBot(cli, conf)
	if conf.HttpConfig != nil && conf.HttpConfig.Enabled {
		server.HttpServer.Run(fmt.Sprintf("%s:%d", conf.HttpConfig.Host, conf.HttpConfig.Port), conf.AccessToken, b)
		for k, v := range conf.HttpConfig.PostUrls {
			server.NewHttpClient().Run(k, v, b)
		}
	}
	if conf.WSConfig != nil && conf.WSConfig.Enabled {
		server.WebsocketServer.Run(fmt.Sprintf("%s:%d", conf.WSConfig.Host, conf.WSConfig.Port), conf.AccessToken, b)
	}
	for _, rc := range conf.ReverseServers {
		server.NewWebsocketClient(rc, conf.AccessToken, b).Run()
	}
	log.Info("资源初始化完成, 开始处理信息.")
	log.Info("アトリは、高性能ですから!")
	cli.OnDisconnected(func(bot *client.QQClient, e *client.ClientDisconnectedEvent) {
		if conf.ReLogin {
			log.Warnf("Bot已离线 (%v)，将在 %v 秒后尝试重连.", e.Message, conf.ReLoginDelay)
			time.Sleep(time.Second * time.Duration(conf.ReLoginDelay))
			rsp, err := cli.Login()
			if err != nil {
				log.Fatalf("重连失败: %v", err)
			}
			if !rsp.Success {
				switch rsp.Error {
				case client.NeedCaptcha:
					log.Fatalf("重连失败: 需要验证码. (验证码处理正在开发中)")
				case client.UnsafeDeviceError:
					log.Fatalf("重连失败: 设备锁")
				default:
					log.Fatalf("重连失败: %v", rsp.ErrorMessage)
				}
			}
			return
		}
		b.Release()
		log.Fatalf("Bot已离线：%v", e.Message)
	})
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	<-c
	b.Release()
}
