package main

import (
	"bufio"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/Mrs4s/go-cqhttp/server"
	"github.com/guonaihong/gout"
	"github.com/tidwall/gjson"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/Mrs4s/MiraiGo/binary"
	"github.com/Mrs4s/MiraiGo/client"
	"github.com/Mrs4s/go-cqhttp/coolq"
	"github.com/Mrs4s/go-cqhttp/global"
	"github.com/getlantern/go-update"
	jsoniter "github.com/json-iterator/go"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/rifflock/lfshook"
	log "github.com/sirupsen/logrus"
	easy "github.com/t-tomalak/logrus-easy-formatter"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func init() {
	log.SetFormatter(&easy.Formatter{
		TimestampFormat: "2006-01-02 15:04:05",
		LogFormat:       "[%time%] [%lvl%]: %msg% \n",
	})
	w, err := rotatelogs.New(path.Join("logs", "%Y-%m-%d.log"), rotatelogs.WithRotationTime(time.Hour*24))
	if err == nil {
		log.SetOutput(io.MultiWriter(os.Stderr, w))
	}
	if !global.PathExists(global.IMAGE_PATH) {
		if err := os.MkdirAll(global.IMAGE_PATH, 0755); err != nil {
			log.Fatalf("创建图片缓存文件夹失败: %v", err)
		}
	}
	if !global.PathExists(global.VOICE_PATH) {
		if err := os.MkdirAll(global.VOICE_PATH, 0755); err != nil {
			log.Fatalf("创建语音缓存文件夹失败: %v", err)
		}
	}
	if !global.PathExists(global.VIDEO_PATH) {
		if err := os.MkdirAll(global.VIDEO_PATH, 0755); err != nil {
			log.Fatalf("创建视频缓存文件夹失败: %v", err)
		}
	}
	if !global.PathExists(global.CACHE_PATH) {
		if err := os.MkdirAll(global.CACHE_PATH, 0755); err != nil {
			log.Fatalf("创建发送图片缓存文件夹失败: %v", err)
		}
	}
	if global.PathExists("cqhttp.json") {
		log.Info("发现 cqhttp.json 将在五秒后尝试导入配置，按 Ctrl+C 取消.")
		log.Warn("警告: 该操作会删除 cqhttp.json 并覆盖 config.hjson 文件.")
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
		if err := goConf.Save("config.hjson"); err != nil {
			log.Fatalf("保存 config.hjson 时出现错误: %v", err)
		}
		_ = os.Remove("cqhttp.json")
	}
}

func main() {
	var byteKey []byte
	var isFastStart bool = false
	arg := os.Args
	if len(arg) > 1 {
		for i := range arg {
			switch arg[i] {
			case "update":
				if len(arg) > i+1 {
					selfUpdate(arg[i+1])
				} else {
					selfUpdate("")
				}
			case "key":
				if len(arg) > i+1 {
					b := []byte(arg[i+1])
					byteKey = b
				}
			case "faststart":
				isFastStart = true
			}
		}
	}

	var conf *global.JsonConfig
	if global.PathExists("config.json") {
		conf = global.Load("config.json")
		_ = conf.Save("config.hjson")
		_ = os.Remove("config.json")
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
			PostMessageFormat: "string",
			Debug:             os.Getenv("DEBUG") == "true",
		}
		if post != "" {
			conf.HttpConfig.PostUrls[post] = os.Getenv("HTTP_SECRET")
		}
	} else {
		conf = global.Load("config.hjson")
	}
	if conf == nil {
		err := global.WriteAllText("config.hjson", global.DefaultConfigWithComments)
		if err != nil {
			log.Fatalf("创建默认配置文件时出现错误: %v", err)
			return
		}
		log.Infof("默认配置文件已生成, 请编辑 config.hjson 后重启程序.")
		time.Sleep(time.Second * 5)
		return
	}
	if conf.Uin == 0 || (conf.Password == "" && conf.PasswordEncrypted == "") {
		log.Warnf("请修改 config.hjson 以添加账号密码.")
		time.Sleep(time.Second * 5)
		return
	}

	// log classified by level
	// Collect all records up to the specified level (default level: warn)
	logLevel := conf.LogLevel
	if logLevel != "" {
		date := time.Now().Format("2006-01-02")
		var logPathMap lfshook.PathMap
		switch conf.LogLevel {
		case "warn":
			logPathMap = lfshook.PathMap{
				log.WarnLevel:  path.Join("logs", date+"-warn.log"),
				log.ErrorLevel: path.Join("logs", date+"-warn.log"),
				log.FatalLevel: path.Join("logs", date+"-warn.log"),
				log.PanicLevel: path.Join("logs", date+"-warn.log"),
			}
		case "error":
			logPathMap = lfshook.PathMap{
				log.ErrorLevel: path.Join("logs", date+"-error.log"),
				log.FatalLevel: path.Join("logs", date+"-error.log"),
				log.PanicLevel: path.Join("logs", date+"-error.log"),
			}
		default:
			logPathMap = lfshook.PathMap{
				log.WarnLevel:  path.Join("logs", date+"-warn.log"),
				log.ErrorLevel: path.Join("logs", date+"-warn.log"),
				log.FatalLevel: path.Join("logs", date+"-warn.log"),
				log.PanicLevel: path.Join("logs", date+"-warn.log"),
			}
		}

		log.AddHook(lfshook.NewHook(
			logPathMap,
			&easy.Formatter{
				TimestampFormat: "2006-01-02 15:04:05",
				LogFormat:       "[%time%] [%lvl%]: %msg% \n",
			},
		))
	}

	log.Info("当前版本:", coolq.Version)
	if conf.Debug {
		log.SetLevel(log.DebugLevel)
		log.Warnf("已开启Debug模式.")
		log.Debugf("开发交流群: 192548878")
		server.Debug = true
		if conf.WebUi == nil || !conf.WebUi.Enabled {
			log.Warnf("警告: 在Debug模式下未启用WebUi服务, 将无法进行性能分析.")
		}
	}
	if !global.PathExists("device.json") {
		log.Warn("虚拟设备信息不存在, 将自动生成随机设备.")
		client.GenRandomDevice()
		_ = ioutil.WriteFile("device.json", client.SystemDeviceInfo.ToJson(), 0644)
		log.Info("已生成设备信息并保存到 device.json 文件.")
	} else {
		log.Info("将使用 device.json 内的设备信息运行Bot.")
		if err := client.SystemDeviceInfo.ReadJson([]byte(global.ReadAllText("device.json"))); err != nil {
			log.Fatalf("加载设备信息失败: %v", err)
		}
	}
	if conf.EncryptPassword && conf.PasswordEncrypted == "" {
		log.Infof("密码加密已启用, 请输入Key对密码进行加密: (Enter 提交)")
		byteKey, _ := terminal.ReadPassword(int(os.Stdin.Fd()))
		key := md5.Sum(byteKey)
		if encrypted := EncryptPwd(conf.Password, key[:]); encrypted != "" {
			conf.Password = ""
			conf.PasswordEncrypted = encrypted
			_ = conf.Save("config.hjson")
		} else {
			log.Warnf("加密时出现问题.")
		}
	}
	if conf.PasswordEncrypted != "" {
		if len(byteKey) == 0 {
			log.Infof("密码加密已启用, 请输入Key对密码进行解密以继续: (Enter 提交)")
			cancel := make(chan struct{}, 1)
			go func() {
				select {
				case <-cancel:
					return
				case <-time.After(time.Second * 45):
					log.Infof("解密key输入超时")
					time.Sleep(3 * time.Second)
					os.Exit(0)
				}
			}()
			byteKey, _ = terminal.ReadPassword(int(os.Stdin.Fd()))
			cancel <- struct{}{}
		} else {
			log.Infof("密码加密已启用, 使用运行时传递的参数进行解密，按 Ctrl+C 取消.")
		}
		key := md5.Sum(byteKey)
		conf.Password = DecryptPwd(conf.PasswordEncrypted, key[:])
	}
	if !isFastStart {
		log.Info("Bot将在5秒后登录并开始信息处理, 按 Ctrl+C 取消.")
		time.Sleep(time.Second * 5)
	}
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
	if global.PathExists("address.txt") {
		log.Infof("检测到 address.txt 文件. 将覆盖目标IP.")
		addr := global.ReadAddrFile("address.txt")
		if len(addr) > 0 {
			cli.SetCustomServer(addr)
		}
		log.Infof("读取到 %v 个自定义地址.", len(addr))
	}
	cli.OnServerUpdated(func(bot *client.QQClient, e *client.ServerUpdatedEvent) bool {
		if !conf.UseSSOAddress {
			log.Infof("收到服务器地址更新通知, 根据配置文件已忽略.")
			return false
		}
		log.Infof("收到服务器地址更新通知, 将在下一次重连时应用. ")
		return true
	})
	if conf.WebUi == nil {
		conf.WebUi = &global.GoCqWebUi{
			Enabled:   true,
			WebInput:  false,
			Host:      "0.0.0.0",
			WebUiPort: 9999,
		}
	}
	if conf.WebUi.WebUiPort <= 0 {
		conf.WebUi.WebUiPort = 9999
	}
	if conf.WebUi.Host == "" {
		conf.WebUi.Host = "127.0.0.1"
	}
	global.Proxy = conf.ProxyRewrite
	b := server.WebServer.Run(fmt.Sprintf("%s:%d", conf.WebUi.Host, conf.WebUi.WebUiPort), cli)
	c := server.Console
	r := server.Restart
	go checkUpdate()
	signal.Notify(c, os.Interrupt, os.Kill)
	select {
	case <-c:
		b.Release()
	case <-r:
		log.Info("正在重启中...")
		defer b.Release()
		restart(arg)
	}
}

func EncryptPwd(pwd string, key []byte) string {
	tea := binary.NewTeaCipher(key)
	if tea == nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(tea.Encrypt([]byte(pwd)))
}

func DecryptPwd(ePwd string, key []byte) string {
	defer func() {
		if pan := recover(); pan != nil {
			log.Fatalf("密码解密失败: %v", pan)
		}
	}()
	encrypted, err := base64.StdEncoding.DecodeString(ePwd)
	if err != nil {
		panic(err)
	}
	tea := binary.NewTeaCipher(key)
	if tea == nil {
		panic("密钥错误")
	}
	return string(tea.Decrypt(encrypted))
}

func checkUpdate() {
	log.Infof("正在检查更新.")
	if coolq.Version == "unknown" {
		log.Warnf("检查更新失败: 使用的 Actions 测试版或自编译版本.")
		return
	}
	var res string
	if err := gout.GET("https://api.github.com/repos/Mrs4s/go-cqhttp/releases").BindBody(&res).Do(); err != nil {
		log.Warnf("检查更新失败: %v", err)
		return
	}
	detail := gjson.Parse(res)
	if len(detail.Array()) < 1 {
		return
	}
	info := detail.Array()[0]
	if global.VersionNameCompare(coolq.Version, info.Get("tag_name").Str) {
		log.Infof("当前有更新的 go-cqhttp 可供更新, 请前往 https://github.com/Mrs4s/go-cqhttp/releases 下载.")
		log.Infof("当前版本: %v 最新版本: %v", coolq.Version, info.Get("tag_name").Str)
		return
	}
	log.Infof("检查更新完成. 当前已运行最新版本.")
}

func selfUpdate(imageUrl string) {
	console := bufio.NewReader(os.Stdin)
	readLine := func() (str string) {
		str, _ = console.ReadString('\n')
		return
	}
	log.Infof("正在检查更新.")
	var res string
	if err := gout.GET("https://api.github.com/repos/Mrs4s/go-cqhttp/releases").BindBody(&res).Do(); err != nil {
		log.Warnf("检查更新失败: %v", err)
		return
	}
	detail := gjson.Parse(res)
	if len(detail.Array()) < 1 {
		return
	}
	info := detail.Array()[0]
	version := info.Get("tag_name").Str
	if coolq.Version != version {
		log.Info("当前最新版本为 ", version)
		log.Warn("是否更新(y/N): ")
		r := strings.TrimSpace(readLine())

		doUpdate := func() {
			log.Info("正在更新,请稍等...")
			url := fmt.Sprintf(
				"%v/Mrs4s/go-cqhttp/releases/download/%v/go-cqhttp-%v-%v-%v",
				func() string {
					if imageUrl != "" {
						return imageUrl
					}
					return "https://github.com"
				}(),
				version,
				version,
				runtime.GOOS,
				runtime.GOARCH,
			)
			if runtime.GOOS == "windows" {
				url = url + ".exe"
			}
			resp, err := http.Get(url)
			if err != nil {
				fmt.Println(err)
				log.Error("更新失败!")
				return
			}
			wc := global.WriteCounter{}
			err, _ = update.New().FromStream(io.TeeReader(resp.Body, &wc))
			fmt.Println()
			if err != nil {
				log.Error("更新失败!")
				return
			}
			log.Info("更新完成！")
		}

		if r == "y" || r == "Y" {
			doUpdate()
		} else {
			log.Warn("已取消更新！")
		}
	}
	log.Info("按 Enter 继续....")
	readLine()
	os.Exit(0)
}

func restart(Args []string) {
	cmd := &exec.Cmd{}
	if runtime.GOOS == "windows" {
		file, err := exec.LookPath(Args[0])
		if err != nil {
			log.Errorf("重启失败:%s", err.Error())
			return
		}
		path, err := filepath.Abs(file)
		if err != nil {
			log.Errorf("重启失败:%s", err.Error())
		}
		Args = append([]string{"/c", "start ", path, "faststart"}, Args[1:]...)
		cmd = &exec.Cmd{
			Path:   "cmd.exe",
			Args:   Args,
			Stderr: os.Stderr,
			Stdout: os.Stdout,
		}
	} else {
		Args = append(Args, "faststart")
		cmd = &exec.Cmd{
			Path:   Args[0],
			Args:   Args,
			Stderr: os.Stderr,
			Stdout: os.Stdout,
		}
	}
	cmd.Start()
}
