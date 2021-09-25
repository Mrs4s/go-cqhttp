package main

import (
	"crypto/aes"
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Mrs4s/MiraiGo/binary"
	"github.com/Mrs4s/MiraiGo/client"
	para "github.com/fumiama/go-hide-param"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/term"

	_ "github.com/Mrs4s/go-cqhttp/modules/mime" // mime检查模块
	_ "github.com/Mrs4s/go-cqhttp/modules/silk" // silk编码模块

	"github.com/Mrs4s/go-cqhttp/coolq"
	"github.com/Mrs4s/go-cqhttp/global"
	"github.com/Mrs4s/go-cqhttp/global/config"
	"github.com/Mrs4s/go-cqhttp/global/terminal"
	"github.com/Mrs4s/go-cqhttp/internal/base"
	"github.com/Mrs4s/go-cqhttp/internal/selfupdate"
	"github.com/Mrs4s/go-cqhttp/server"
)

var (
	conf        *config.Config
	isFastStart = false

	// 允许通过配置文件设置的状态列表
	allowStatus = [...]client.UserOnlineStatus{
		client.StatusOnline, client.StatusAway, client.StatusInvisible, client.StatusBusy,
		client.StatusListening, client.StatusConstellation, client.StatusWeather, client.StatusMeetSpring,
		client.StatusTimi, client.StatusEatChicken, client.StatusLoving, client.StatusWangWang, client.StatusCookedRice,
		client.StatusStudy, client.StatusStayUp, client.StatusPlayBall, client.StatusSignal, client.StatusStudyOnline,
		client.StatusGaming, client.StatusVacationing, client.StatusWatchingTV, client.StatusFitness,
	}
)

func main() {
	c := flag.String("c", config.DefaultConfigFile, "configuration filename")
	d := flag.Bool("d", false, "running as a daemon")
	h := flag.Bool("h", false, "this help")
	wd := flag.String("w", "", "cover the working directory")
	debug := flag.Bool("D", false, "debug mode")
	flag.Parse()
	// todo: maybe move flag to internal/base?

	switch {
	case *h:
		help()
	case *d:
		server.Daemon()
	case *wd != "":
		resetWorkDir(*wd)
	}

	// 通过-c 参数替换 配置文件路径
	config.DefaultConfigFile = *c
	conf = config.Get()
	if *debug {
		conf.Output.Debug = true
	}
	base.Parse()
	if base.Debug {
		log.SetReportCaller(true)
	}

	rotateOptions := []rotatelogs.Option{
		rotatelogs.WithRotationTime(time.Hour * 24),
	}

	if conf.Output.LogAging > 0 {
		rotateOptions = append(rotateOptions, rotatelogs.WithMaxAge(time.Hour*24*time.Duration(conf.Output.LogAging)))
	} else {
		rotateOptions = append(rotateOptions, rotatelogs.WithMaxAge(time.Hour*24*365*10))
	}
	if conf.Output.LogForceNew {
		rotateOptions = append(rotateOptions, rotatelogs.ForceNewFile())
	}

	w, err := rotatelogs.New(path.Join("logs", "%Y-%m-%d.log"), rotateOptions...)
	if err != nil {
		log.Errorf("rotatelogs init err: %v", err)
		panic(err)
	}

	log.AddHook(global.NewLocalHook(w, global.LogFormat{}, global.GetLogLevel(conf.Output.LogLevel)...))

	mkCacheDir := func(path string, _type string) {
		if !global.PathExists(path) {
			if err := os.MkdirAll(path, 0o755); err != nil {
				log.Fatalf("创建%s缓存文件夹失败: %v", _type, err)
			}
		}
	}
	mkCacheDir(global.ImagePath, "图片")
	mkCacheDir(global.VoicePath, "语音")
	mkCacheDir(global.VideoPath, "视频")
	mkCacheDir(global.CachePath, "发送图片")

	var byteKey []byte
	arg := os.Args
	if len(arg) > 1 {
		for i := range arg {
			switch arg[i] {
			case "update":
				if len(arg) > i+1 {
					selfupdate.SelfUpdate(arg[i+1])
				} else {
					selfupdate.SelfUpdate("")
				}
			case "key":
				p := i + 1
				if len(arg) > p {
					byteKey = []byte(arg[p])
					para.Hide(p)
				}
			case "faststart":
				isFastStart = true
			}
		}
	}
	if terminal.RunningByDoubleClick() && !isFastStart {
		err := terminal.NoMoreDoubleClick()
		if err != nil {
			log.Errorf("遇到错误: %v", err)
			time.Sleep(time.Second * 5)
		}
		return
	}

	if (conf.Account.Uin == 0 || (conf.Account.Password == "" && !conf.Account.Encrypt)) && !global.PathExists("session.token") {
		log.Warn("账号密码未配置, 将使用二维码登录.")
		if !isFastStart {
			log.Warn("将在 5秒 后继续.")
			time.Sleep(time.Second * 5)
		}
	}

	log.Info("当前版本:", base.Version)
	if base.Debug {
		log.SetLevel(log.DebugLevel)
		log.Warnf("已开启Debug模式.")
		log.Debugf("开发交流群: 192548878")
	}
	log.Info("用户交流群: 721829413")
	if !global.PathExists("device.json") {
		log.Warn("虚拟设备信息不存在, 将自动生成随机设备.")
		client.GenRandomDevice()
		_ = os.WriteFile("device.json", client.SystemDeviceInfo.ToJson(), 0o644)
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
			base.PasswordHash = md5.Sum([]byte(conf.Account.Password))
			_ = os.WriteFile("password.encrypt", []byte(PasswordHashEncrypt(base.PasswordHash[:], byteKey)), 0o644)
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
			copy(base.PasswordHash[:], ph)
		}
	} else if len(conf.Account.Password) > 0 {
		base.PasswordHash = md5.Sum([]byte(conf.Account.Password))
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
		case client.QiDian:
			return "企点"
		}
		return "未知"
	}())
	cli = newClient()
	isQRCodeLogin := (conf.Account.Uin == 0 || len(conf.Account.Password) == 0) && !conf.Account.Encrypt
	isTokenLogin := false
	saveToken := func() {
		base.AccountToken = cli.GenToken()
		_ = os.WriteFile("session.token", base.AccountToken, 0o644)
	}
	if global.PathExists("session.token") {
		token, err := os.ReadFile("session.token")
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
						log.Infof("缓存已删除.")
						os.Exit(0)
					}
				}
			}
			if err = cli.TokenLogin(token); err != nil {
				_ = os.Remove("session.token")
				log.Warnf("恢复会话失败: %v , 尝试使用正常流程登录.", err)
				time.Sleep(time.Second)
				cli.Disconnect()
				cli.Release()
				cli = newClient()
			} else {
				isTokenLogin = true
			}
		}
	}
	if conf.Account.Uin != 0 && base.PasswordHash != [16]byte{} {
		cli.Uin = conf.Account.Uin
		cli.PasswordMd5 = base.PasswordHash
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
				log.Warnf("未启用自动重连, 将退出.")
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
			if cli.Online {
				log.Infof("登录已完成")
				break
			}
			log.Warnf("尝试重连...")
			err := cli.TokenLogin(base.AccountToken)
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
				log.Warn("读取正向Websocket配置失败 :", err)
			} else {
				go server.RunWebSocketServer(bot, sc)
			}
		}
		if c, ok := m["ws-reverse"]; ok {
			rc := new(config.WebsocketReverse)
			if err := c.Decode(rc); err != nil {
				log.Warn("读取反向Websocket配置失败 :", err)
			} else {
				go server.RunWebSocketClient(bot, rc)
			}
		}
		if p, ok := m["pprof"]; ok {
			pc := new(config.PprofServer)
			if err := p.Decode(pc); err != nil {
				log.Warn("读取pprof配置失败 :", err)
			} else {
				go server.RunPprofServer(pc)
			}
		}
		if p, ok := m["lambda"]; ok {
			lc := new(config.LambdaServer)
			if err := p.Decode(lc); err != nil {
				log.Warn("读取pprof配置失败 :", err)
			} else {
				go server.RunLambdaClient(bot, lc)
			}
		}
	}
	log.Info("资源初始化完成, 开始处理信息.")
	log.Info("アトリは、高性能ですから!")

	go selfupdate.CheckUpdate()

	<-global.SetupMainSignalHandler()
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

// help cli命令行-h的帮助提示
func help() {
	fmt.Printf(`go-cqhttp service
version: %s

Usage:

server [OPTIONS]

Options:
`, base.Version)

	flag.PrintDefaults()
	os.Exit(0)
}

func resetWorkDir(wd string) {
	args := make([]string, 0, len(os.Args))
	for i := 1; i < len(os.Args); i++ {
		if os.Args[i] == "-w" {
			i++ // skip value field
		} else if !strings.HasPrefix(os.Args[i], "-w") {
			args = append(args, os.Args[i])
		}
	}
	p, _ := filepath.Abs(os.Args[0])
	proc := exec.Command(p, args...)
	proc.Stdin = os.Stdin
	proc.Stdout = os.Stdout
	proc.Stderr = os.Stderr
	proc.Dir = wd
	err := proc.Run()
	if err != nil {
		panic(err)
	}
	os.Exit(0)
}

func newClient() *client.QQClient {
	c := client.NewClientEmpty()
	c.OnServerUpdated(func(bot *client.QQClient, e *client.ServerUpdatedEvent) bool {
		if !conf.Account.UseSSOAddress {
			log.Infof("收到服务器地址更新通知, 根据配置文件已忽略.")
			return false
		}
		log.Infof("收到服务器地址更新通知, 将在下一次重连时应用. ")
		return true
	})
	if global.PathExists("address.txt") {
		log.Infof("检测到 address.txt 文件. 将覆盖目标IP.")
		addr := global.ReadAddrFile("address.txt")
		if len(addr) > 0 {
			c.SetCustomServer(addr)
		}
		log.Infof("读取到 %v 个自定义地址.", len(addr))
	}
	c.OnLog(func(c *client.QQClient, e *client.LogEvent) {
		switch e.Type {
		case "INFO":
			log.Info("Protocol -> " + e.Message)
		case "ERROR":
			log.Error("Protocol -> " + e.Message)
		case "DEBUG":
			log.Debug("Protocol -> " + e.Message)
		}
	})
	return c
}
