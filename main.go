package main

import (
	"crypto/aes"
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Mrs4s/MiraiGo/binary"

	"github.com/Mrs4s/go-cqhttp/coolq"
	"github.com/Mrs4s/go-cqhttp/global"
	"github.com/Mrs4s/go-cqhttp/global/config"
	"github.com/Mrs4s/go-cqhttp/global/terminal"
	"github.com/Mrs4s/go-cqhttp/global/update"
	"github.com/Mrs4s/go-cqhttp/server"

	"github.com/Mrs4s/MiraiGo/client"
	"github.com/guonaihong/gout"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	log "github.com/sirupsen/logrus"
	easy "github.com/t-tomalak/logrus-easy-formatter"
	"github.com/tidwall/gjson"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/term"
)

var (
	conf        *config.Config
	isFastStart = false
	c           string
	d           bool
	h           bool

	// 允许通过配置文件设置的状态列表
	allowStatus = [...]client.UserOnlineStatus{
		client.StatusOnline, client.StatusAway, client.StatusInvisible, client.StatusBusy,
		client.StatusListening, client.StatusConstellation, client.StatusWeather, client.StatusMeetSpring,
		client.StatusTimi, client.StatusEatChicken, client.StatusLoving, client.StatusWangWang, client.StatusCookedRice,
		client.StatusStudy, client.StatusStayUp, client.StatusPlayBall, client.StatusSignal, client.StatusStudyOnline,
		client.StatusGaming, client.StatusVacationing, client.StatusWatchingTV, client.StatusFitness,
	}
)

func init() {
	var debug bool
	flag.StringVar(&c, "c", config.DefaultConfigFile, "configuration filename default is config.hjson")
	flag.BoolVar(&d, "d", false, "running as a daemon")
	flag.BoolVar(&debug, "D", false, "debug mode")
	flag.BoolVar(&h, "h", false, "this help")
	flag.Parse()

	// 通过-c 参数替换 配置文件路径
	config.DefaultConfigFile = c
	logFormatter := &easy.Formatter{
		TimestampFormat: "2006-01-02 15:04:05",
		LogFormat:       "[%time%] [%lvl%]: %msg% \n",
	}
	w, err := rotatelogs.New(path.Join("logs", "%Y-%m-%d.log"), rotatelogs.WithRotationTime(time.Hour*24))
	if err != nil {
		log.Errorf("rotatelogs init err: %v", err)
		panic(err)
	}

	conf = config.Get()
	if conf == nil {
		_ = os.WriteFile("config.yml", []byte(config.DefaultConfig), 0644)
		log.Error("未找到配置文件，默认配置文件已生成!")
		readLine()
		os.Exit(0)
	}

	if debug {
		conf.Output.Debug = true
	}
	// 在debug模式下,将在标准输出中打印当前执行行数
	if conf.Output.Debug {
		log.SetReportCaller(true)
	}

	log.AddHook(global.NewLocalHook(w, logFormatter, global.GetLogLevel(conf.Output.LogLevel)...))

	if !global.PathExists(global.ImagePath) {
		if err := os.MkdirAll(global.ImagePath, 0755); err != nil {
			log.Fatalf("创建图片缓存文件夹失败: %v", err)
		}
	}
	if !global.PathExists(global.VoicePath) {
		if err := os.MkdirAll(global.VoicePath, 0755); err != nil {
			log.Fatalf("创建语音缓存文件夹失败: %v", err)
		}
	}
	if !global.PathExists(global.VideoPath) {
		if err := os.MkdirAll(global.VideoPath, 0755); err != nil {
			log.Fatalf("创建视频缓存文件夹失败: %v", err)
		}
	}
	if !global.PathExists(global.CachePath) {
		if err := os.MkdirAll(global.CachePath, 0755); err != nil {
			log.Fatalf("创建发送图片缓存文件夹失败: %v", err)
		}
	}
}

func main() {
	if h {
		help()
	}
	if d {
		server.Daemon()
	}
	var byteKey []byte
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
					byteKey = []byte(arg[i+1])
				}
			case "faststart":
				isFastStart = true
			}
		}
	}
	if terminal.RunningByDoubleClick() && !isFastStart {
		log.Warning("警告: 强烈不推荐通过双击直接运行本程序, 这将导致一些非预料的后果.")
		log.Warning("将等待10s后启动")
		time.Sleep(time.Second * 10)
	}

	if (conf.Account.Uin == 0 || (conf.Account.Password == "" && !conf.Account.Encrypt)) && !global.PathExists("session.token") {
		log.Warn("账号密码未配置, 将使用二维码登录.")
		if !isFastStart {
			log.Warn("将在 5秒 后继续.")
			time.Sleep(time.Second * 5)
		}
	}

	log.Info("当前版本:", coolq.Version)
	if conf.Output.Debug {
		log.SetLevel(log.DebugLevel)
		log.Warnf("已开启Debug模式.")
		log.Debugf("开发交流群: 192548878")
	}
	log.Info("用户交流群: 721829413")
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

	if conf.Account.Encrypt {
		if !global.PathExists("password.encrypt") {
			if conf.Account.Password == "" {
				log.Error("无法进行加密，请在配置文件中的添加密码后重新启动.")
				readLine()
				os.Exit(0)
			}
			log.Infof("密码加密已启用, 请输入Key对密码进行加密: (Enter 提交)")
			byteKey, _ = term.ReadPassword(int(os.Stdin.Fd()))
			global.PasswordHash = md5.Sum([]byte(conf.Account.Password))
			_ = os.WriteFile("password.encrypt", []byte(PasswordHashEncrypt(global.PasswordHash[:], byteKey)), 0644)
			log.Info("密码已加密，为了您的账号安全，请删除配置文件中的密码后重新启动.")
			readLine()
			os.Exit(0)
		} else {
			if conf.Account.Password != "" {
				log.Error("密码已加密，为了您的账号安全，请删除配置文件中的密码后重新启动.")
				readLine()
				os.Exit(0)
			}

			if len(byteKey) == 0 {
				log.Infof("密码加密已启用, 请输入Key对密码进行解密以继续: (Enter 提交)")
				cancel := make(chan struct{}, 1)
				state, _ := term.GetState(int(os.Stdin.Fd()))
				go func() {
					select {
					case <-cancel:
						return
					case <-time.After(time.Second * 45):
						log.Infof("解密key输入超时")
						time.Sleep(3 * time.Second)
						_ = term.Restore(int(os.Stdin.Fd()), state)
						os.Exit(0)
					}
				}()
				byteKey, _ = term.ReadPassword(int(os.Stdin.Fd()))
				cancel <- struct{}{}
			} else {
				log.Infof("密码加密已启用, 使用运行时传递的参数进行解密，按 Ctrl+C 取消.")
			}

			encrypt, _ := os.ReadFile("password.encrypt")
			ph, err := PasswordHashDecrypt(string(encrypt), byteKey)
			if err != nil {
				log.Fatalf("加密存储的密码损坏，请尝试重新配置密码")
			}
			copy(global.PasswordHash[:], ph)
		}
	} else {
		global.PasswordHash = md5.Sum([]byte(conf.Account.Password))
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
	cli = client.NewClientEmpty()
	if conf.Account.Uin != 0 && global.PasswordHash != [16]byte{} {
		cli.Uin = conf.Account.Uin
		cli.PasswordMd5 = global.PasswordHash
	}
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
		if !conf.Account.UseSSOAddress {
			log.Infof("收到服务器地址更新通知, 根据配置文件已忽略.")
			return false
		}
		log.Infof("收到服务器地址更新通知, 将在下一次重连时应用. ")
		return true
	})
	global.Proxy = conf.Message.ProxyRewrite
	isQRCodeLogin := (conf.Account.Uin == 0 || len(conf.Account.Password) == 0) && !conf.Account.Encrypt
	isTokenLogin := false
	saveToken := func() {
		global.AccountToken = cli.GenToken()
		_ = ioutil.WriteFile("session.token", global.AccountToken, 0677)
	}
	if global.PathExists("session.token") {
		token, err := ioutil.ReadFile("session.token")
		if err == nil {
			if conf.Account.Uin != 0 {
				r := binary.NewReader(token)
				cu := r.ReadInt64()
				if cu != conf.Account.Uin {
					log.Warnf("警告: 配置文件内的QQ号 (%v) 与缓存内的QQ号 (%v) 不相同", conf.Account.Uin, cu)
					log.Warnf("1. 使用会话缓存继续.")
					log.Warnf("2. 删除会话缓存并重启.")
					log.Warnf("请选择: (5秒后自动选1)")
					text := readLineTimeout(time.Second*5, "1")
					if text == "2" {
						_ = os.Remove("session.token")
						os.Exit(0)
					}
				}
			}
			if err = cli.TokenLogin(token); err != nil {
				_ = os.Remove("session.token")
				log.Warnf("恢复会话失败: %v , 尝试使用正常流程登录.", err)
				time.Sleep(time.Second)
			} else {
				isTokenLogin = true
			}
		}
	}
	if !isTokenLogin {
		if !isQRCodeLogin {
			if err := commonLogin(); err != nil {
				log.Fatalf("登录时发生致命错误: %v", err)
			}
		} else {
			if err := qrcodeLogin(); err != nil {
				log.Fatalf("登录时发生致命错误: %v", err)
			}
		}
	}
	var times uint = 1 // 重试次数
	var reLoginLock sync.Mutex
	cli.OnDisconnected(func(q *client.QQClient, e *client.ClientDisconnectedEvent) {
		reLoginLock.Lock()
		defer reLoginLock.Unlock()
		times = 1
		if cli.Online {
			return
		}
		log.Warnf("Bot已离线: %v", e.Message)
		time.Sleep(time.Second * time.Duration(conf.Account.ReLogin.Delay))
		for {
			if conf.Account.ReLogin.Disabled {
				os.Exit(1)
			}
			if times > conf.Account.ReLogin.MaxTimes && conf.Account.ReLogin.MaxTimes != 0 {
				log.Fatalf("Bot重连次数超过限制, 停止")
			}
			times++
			if conf.Account.ReLogin.Interval > 0 {
				log.Warnf("将在 %v 秒后尝试重连. 重连次数：%v/%v", conf.Account.ReLogin.Interval, times, conf.Account.ReLogin.MaxTimes)
				time.Sleep(time.Second * time.Duration(conf.Account.ReLogin.Interval))
			} else {
				time.Sleep(time.Second)
			}
			log.Warnf("尝试重连...")
			err := cli.TokenLogin(global.AccountToken)
			if err == nil {
				saveToken()
				return
			}
			log.Warnf("快速重连失败: %v", err)
			if isQRCodeLogin {
				log.Fatalf("快速重连失败, 扫码登录无法恢复会话.")
			}
			log.Warnf("快速重连失败, 尝试普通登录. 这可能是因为其他端强行T下线导致的.")
			time.Sleep(time.Second)
			if err := commonLogin(); err != nil {
				log.Errorf("登录时发生致命错误: %v", err)
			} else {
				saveToken()
				break
			}
		}
	})
	saveToken()
	cli.AllowSlider = true
	log.Infof("登录成功 欢迎使用: %v", cli.Nickname)
	log.Info("开始加载好友列表...")
	global.Check(cli.ReloadFriendList(), true)
	log.Infof("共加载 %v 个好友.", len(cli.FriendList))
	log.Infof("开始加载群列表...")
	global.Check(cli.ReloadGroupList(), true)
	log.Infof("共加载 %v 个群.", len(cli.GroupList))
	if conf.Account.Status >= int32(len(allowStatus)) || conf.Account.Status < 0 {
		conf.Account.Status = 0
	}
	cli.SetOnlineStatus(allowStatus[int(conf.Account.Status)])
	bot := coolq.NewQQBot(cli, conf)
	_ = bot.Client
	if conf.Message.PostFormat != "string" && conf.Message.PostFormat != "array" {
		log.Warnf("post-format 配置错误, 将自动使用 string")
		coolq.SetMessageFormat("string")
	} else {
		coolq.SetMessageFormat(conf.Message.PostFormat)
	}
	log.Info("正在加载事件过滤器.")
	coolq.IgnoreInvalidCQCode = conf.Message.IgnoreInvalidCQCode
	coolq.SplitURL = conf.Message.FixURL
	coolq.ForceFragmented = conf.Message.ForceFragment
	coolq.RemoveReplyAt = conf.Message.RemoveReplyAt
	coolq.ExtraReplyData = conf.Message.ExtraReplyData
	for _, m := range conf.Servers {
		if h, ok := m["http"]; ok {
			hc := new(config.HTTPServer)
			if err := h.Decode(hc); err != nil {
				log.Warn("读取http配置失败 :", err)
			} else {
				go server.RunHTTPServerAndClients(bot, hc)
			}
		}
		if s, ok := m["ws"]; ok {
			sc := new(config.WebsocketServer)
			if err := s.Decode(sc); err != nil {
				log.Warn("读取http配置失败 :", err)
			} else {
				go server.RunWebSocketServer(bot, sc)
			}
		}
		if c, ok := m["ws-reverse"]; ok {
			rc := new(config.WebsocketReverse)
			if err := c.Decode(rc); err != nil {
				log.Warn("读取http配置失败 :", err)
			} else {
				go server.RunWebSocketClient(bot, rc)
			}
		}
		if p, ok := m["pprof"]; ok {
			pc := new(config.PprofServer)
			if err := p.Decode(pc); err != nil {
				log.Warn("读取http配置失败 :", err)
			} else {
				go server.RunPprofServer(pc)
			}
		}
	}
	log.Info("资源初始化完成, 开始处理信息.")
	log.Info("アトリは、高性能ですから!")
	c := make(chan os.Signal, 1)
	go checkUpdate()
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
}

// PasswordHashEncrypt 使用key加密给定passwordHash
func PasswordHashEncrypt(passwordHash []byte, key []byte) string {
	if len(passwordHash) != 16 {
		panic("密码加密参数错误")
	}

	key = pbkdf2.Key(key, key, 114514, 32, sha1.New)

	cipher, _ := aes.NewCipher(key)
	result := make([]byte, 16)
	cipher.Encrypt(result, passwordHash)

	return hex.EncodeToString(result)
}

// PasswordHashDecrypt 使用key解密给定passwordHash
func PasswordHashDecrypt(encryptedPasswordHash string, key []byte) ([]byte, error) {
	ciphertext, err := hex.DecodeString(encryptedPasswordHash)
	if err != nil {
		return nil, err
	}

	key = pbkdf2.Key(key, key, 114514, 32, sha1.New)

	cipher, _ := aes.NewCipher(key)
	result := make([]byte, 16)
	cipher.Decrypt(result, ciphertext)

	return result, nil
}

func checkUpdate() {
	log.Infof("正在检查更新.")
	if coolq.Version == "(devel)" {
		log.Warnf("检查更新失败: 使用的 Actions 测试版或自编译版本.")
		return
	}
	var res string
	if err := gout.GET("https://api.github.com/repos/Mrs4s/go-cqhttp/releases/latest").BindBody(&res).Do(); err != nil {
		log.Warnf("检查更新失败: %v", err)
		return
	}
	info := gjson.Parse(res)
	if global.VersionNameCompare(coolq.Version, info.Get("tag_name").Str) {
		log.Infof("当前有更新的 go-cqhttp 可供更新, 请前往 https://github.com/Mrs4s/go-cqhttp/releases 下载.")
		log.Infof("当前版本: %v 最新版本: %v", coolq.Version, info.Get("tag_name").Str)
		return
	}
	log.Infof("检查更新完成. 当前已运行最新版本.")
}

func selfUpdate(imageURL string) {
	log.Infof("正在检查更新.")
	var res string
	if err := gout.GET("https://api.github.com/repos/Mrs4s/go-cqhttp/releases/latest").BindBody(&res).Do(); err != nil {
		log.Warnf("检查更新失败: %v", err)
		return
	}
	info := gjson.Parse(res)
	version := info.Get("tag_name").Str
	if coolq.Version != version {
		log.Info("当前最新版本为 ", version)
		log.Warn("是否更新(y/N): ")
		r := strings.TrimSpace(readLine())
		if r != "y" && r != "Y" {
			log.Warn("已取消更新！")
		} else {
			log.Info("正在更新,请稍等...")
			url := fmt.Sprintf(
				"%v/Mrs4s/go-cqhttp/releases/download/%v/go-cqhttp_%v_%v",
				func() string {
					if imageURL != "" {
						return imageURL
					}
					return "https://github.com"
				}(),
				version, runtime.GOOS, func() string {
					if runtime.GOARCH == "arm" {
						return "armv7"
					}
					return runtime.GOARCH
				}(),
			)
			if runtime.GOOS == "windows" {
				url += ".zip"
			} else {
				url += ".tar.gz"
			}
			update.Update(url)
		}
	} else {
		log.Info("当前版本已经是最新版本!")
	}
	log.Info("按 Enter 继续....")
	readLine()
	os.Exit(0)
}

/*
func restart(args []string) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		file, err := exec.LookPath(args[0])
		if err != nil {
			log.Errorf("重启失败:%s", err.Error())
			return
		}
		path, err := filepath.Abs(file)
		if err != nil {
			log.Errorf("重启失败:%s", err.Error())
		}
		args = append([]string{"/c", "start ", path, "faststart"}, args[1:]...)
		cmd = &exec.Cmd{
			Path:   "cmd.exe",
			Args:   args,
			Stderr: os.Stderr,
			Stdout: os.Stdout,
		}
	} else {
		args = append(args, "faststart")
		cmd = &exec.Cmd{
			Path:   args[0],
			Args:   args,
			Stderr: os.Stderr,
			Stdout: os.Stdout,
		}
	}
	_ = cmd.Start()
}
*/

// help cli命令行-h的帮助提示
func help() {
	fmt.Printf(`go-cqhttp service
version: %s

Usage:

server [OPTIONS]

Options:
`, coolq.Version)

	flag.PrintDefaults()
	os.Exit(0)
}
