package main

import (
	"bufio"
	"crypto/aes"
	"crypto/md5"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
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
	"syscall"
	"time"

	"github.com/Mrs4s/go-cqhttp/server"
	"github.com/guonaihong/gout"
	"github.com/tidwall/gjson"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/term"

	"github.com/Mrs4s/MiraiGo/binary"
	"github.com/Mrs4s/MiraiGo/client"
	"github.com/Mrs4s/go-cqhttp/coolq"
	"github.com/Mrs4s/go-cqhttp/global"
	jsoniter "github.com/json-iterator/go"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	log "github.com/sirupsen/logrus"
	easy "github.com/t-tomalak/logrus-easy-formatter"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary
var conf *global.JSONConfig
var isFastStart = false

func init() {
	if global.PathExists("cqhttp.json") {
		log.Info("发现 cqhttp.json 将在五秒后尝试导入配置，按 Ctrl+C 取消.")
		log.Warn("警告: 该操作会删除 cqhttp.json 并覆盖 config.hjson 文件.")
		time.Sleep(time.Second * 5)
		conf := global.CQHTTPAPIConfig{}
		if err := json.Unmarshal([]byte(global.ReadAllText("cqhttp.json")), &conf); err != nil {
			log.Fatalf("读取文件 cqhttp.json 失败: %v", err)
		}
		goConf := global.DefaultConfig()
		goConf.AccessToken = conf.AccessToken
		goConf.HTTPConfig.Host = conf.Host
		goConf.HTTPConfig.Port = conf.Port
		goConf.WSConfig.Host = conf.WSHost
		goConf.WSConfig.Port = conf.WSPort
		if conf.PostURL != "" {
			goConf.HTTPConfig.PostUrls[conf.PostURL] = conf.Secret
		}
		if conf.UseWsReverse {
			goConf.ReverseServers[0].Enabled = true
			goConf.ReverseServers[0].ReverseURL = conf.WSReverseURL
			goConf.ReverseServers[0].ReverseAPIURL = conf.WSReverseAPIURL
			goConf.ReverseServers[0].ReverseEventURL = conf.WSReverseEventURL
			goConf.ReverseServers[0].ReverseReconnectInterval = conf.WSReverseReconnectInterval
		}
		if err := goConf.Save("config.hjson"); err != nil {
			log.Fatalf("保存 config.hjson 时出现错误: %v", err)
		}
		_ = os.Remove("cqhttp.json")
	}

	conf = getConfig()
	if conf == nil {
		os.Exit(1)
	}

	logFormatter := &easy.Formatter{
		TimestampFormat: "2006-01-02 15:04:05",
		LogFormat:       "[%time%] [%lvl%]: %msg% \n",
	}
	w, err := rotatelogs.New(path.Join("logs", "%Y-%m-%d.log"), rotatelogs.WithRotationTime(time.Hour*24))
	if err != nil {
		log.Errorf("rotatelogs init err: %v", err)
		panic(err)
	}

	// 在debug模式下,将在标准输出中打印当前执行行数
	if conf.Debug {
		log.SetReportCaller(true)
	}

	log.AddHook(global.NewLocalHook(w, logFormatter, global.GetLogLevel(conf.LogLevel)...))

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
					b := []byte(arg[i+1])
					byteKey = b
				}
			case "faststart":
				isFastStart = true
			}
		}
	}

	if conf.Uin == 0 || (conf.Password == "" && conf.PasswordEncrypted == "") {
		log.Warnf("请修改 config.hjson 以添加账号密码.")
		if !isFastStart {
			time.Sleep(time.Second * 5)
		}
		return
	}

	log.Info("当前版本:", coolq.Version)
	if conf.Debug {
		log.SetLevel(log.DebugLevel)
		log.Warnf("已开启Debug模式.")
		log.Debugf("开发交流群: 192548878")
		server.Debug = true
		if conf.WebUI == nil || !conf.WebUI.Enabled {
			log.Warnf("警告: 在Debug模式下未启用WebUi服务, 将无法进行性能分析.")
		}
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
	if conf.EncryptPassword && conf.PasswordEncrypted == "" {
		log.Infof("密码加密已启用, 请输入Key对密码进行加密: (Enter 提交)")
		byteKey, _ = term.ReadPassword(int(os.Stdin.Fd()))
		global.PasswordHash = md5.Sum([]byte(conf.Password))
		conf.Password = ""
		conf.PasswordEncrypted = "AES:" + PasswordHashEncrypt(global.PasswordHash[:], byteKey)
		_ = conf.Save("config.hjson")
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
			byteKey, _ = term.ReadPassword(int(os.Stdin.Fd()))
			cancel <- struct{}{}
		} else {
			log.Infof("密码加密已启用, 使用运行时传递的参数进行解密，按 Ctrl+C 取消.")
		}

		//升级客户端密码加密方案，MD5+TEA 加密密码 -> PBKDF2+AES 加密 MD5
		//升级后的 PasswordEncrypted 字符串以"AES:"开始，其后为 Hex 编码的16字节加密 MD5
		if !strings.HasPrefix(conf.PasswordEncrypted, "AES:") {
			password := OldPasswordDecrypt(conf.PasswordEncrypted, byteKey)
			passwordHash := md5.Sum([]byte(password))
			newPasswordHash := PasswordHashEncrypt(passwordHash[:], byteKey)
			conf.PasswordEncrypted = "AES:" + newPasswordHash
			_ = conf.Save("config.hjson")
			log.Debug("密码加密方案升级完成")
		}

		ph, err := PasswordHashDecrypt(conf.PasswordEncrypted[4:], byteKey)
		if err != nil {
			log.Fatalf("加密存储的密码损坏，请尝试重新配置密码")
		}
		copy(global.PasswordHash[:], ph)
	} else {
		global.PasswordHash = md5.Sum([]byte(conf.Password))
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
	cli := client.NewClientMd5(conf.Uin, global.PasswordHash)
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
	if conf.WebUI == nil {
		conf.WebUI = &global.GoCQWebUI{
			Enabled:   true,
			WebInput:  false,
			Host:      "0.0.0.0",
			WebUIPort: 9999,
		}
	}
	if conf.WebUI.WebUIPort <= 0 {
		conf.WebUI.WebUIPort = 9999
	}
	if conf.WebUI.Host == "" {
		conf.WebUI.Host = "127.0.0.1"
	}
	global.Proxy = conf.ProxyRewrite
	b := server.WebServer.Run(fmt.Sprintf("%s:%d", conf.WebUI.Host, conf.WebUI.WebUIPort), cli)
	c := server.Console
	r := server.Restart
	go checkUpdate()
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	select {
	case <-c:
		b.Release()
	case <-r:
		log.Info("正在重启中...")
		defer b.Release()
		restart(arg)
	}
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

// OldPasswordDecrypt 使用key解密老password，仅供兼容使用
func OldPasswordDecrypt(encryptedPassword string, key []byte) string {
	defer func() {
		if pan := recover(); pan != nil {
			log.Fatalf("密码解密失败: %v", pan)
		}
	}()
	encKey := md5.Sum(key)
	encrypted, err := base64.StdEncoding.DecodeString(encryptedPassword)
	if err != nil {
		panic(err)
	}
	tea := binary.NewTeaCipher(encKey[:])
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

func selfUpdate(imageURL string) {
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
					if imageURL != "" {
						return imageURL
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
			err, _ = global.UpdateFromStream(io.TeeReader(resp.Body, &wc))
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
	} else {
		log.Info("当前版本已经是最新版本!")
	}
	log.Info("按 Enter 继续....")
	readLine()
	os.Exit(0)
}

func restart(Args []string) {
	var cmd *exec.Cmd
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
	_ = cmd.Start()
}

func getConfig() *global.JSONConfig {
	var conf *global.JSONConfig
	if global.PathExists("config.json") {
		conf = global.LoadConfig("config.json")
		_ = conf.Save("config.hjson")
		_ = os.Remove("config.json")
	} else if os.Getenv("UIN") != "" {
		log.Infof("将从环境变量加载配置.")
		uin, _ := strconv.ParseInt(os.Getenv("UIN"), 10, 64)
		pwd := os.Getenv("PASS")
		post := os.Getenv("HTTP_POST")
		conf = &global.JSONConfig{
			Uin:      uin,
			Password: pwd,
			HTTPConfig: &global.GoCQHTTPConfig{
				Enabled:  true,
				Host:     "0.0.0.0",
				Port:     5700,
				PostUrls: map[string]string{},
			},
			WSConfig: &global.GoCQWebSocketConfig{
				Enabled: true,
				Host:    "0.0.0.0",
				Port:    6700,
			},
			PostMessageFormat: "string",
			Debug:             os.Getenv("DEBUG") == "true",
		}
		if post != "" {
			conf.HTTPConfig.PostUrls[post] = os.Getenv("HTTP_SECRET")
		}
	} else {
		conf = global.LoadConfig("config.hjson")
	}
	if conf == nil {
		err := global.WriteAllText("config.hjson", global.DefaultConfigWithComments)
		if err != nil {
			log.Fatalf("创建默认配置文件时出现错误: %v", err)
			return nil
		}
		log.Infof("默认配置文件已生成, 请编辑 config.hjson 后重启程序.")
		if !isFastStart {
			time.Sleep(time.Second * 5)
		}
		return nil
	}
	return conf
}
