// Package gocq 程序的主体部分
package qq

import (
	"crypto/aes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"github.com/Mrs4s/MiraiGo/client"
	"github.com/Mrs4s/go-cqhttp/db"
	"github.com/Mrs4s/go-cqhttp/global"
	"github.com/Mrs4s/go-cqhttp/internal/base"
	"github.com/Mrs4s/go-cqhttp/internal/cache"
	"github.com/kataras/iris/v12"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/pbkdf2"
	"os"
	"path"
	"time"
)

// 允许通过配置文件设置的状态列表
var allowStatus = [...]client.UserOnlineStatus{
	client.StatusOnline, client.StatusAway, client.StatusInvisible, client.StatusBusy,
	client.StatusListening, client.StatusConstellation, client.StatusWeather, client.StatusMeetSpring,
	client.StatusTimi, client.StatusEatChicken, client.StatusLoving, client.StatusWangWang, client.StatusCookedRice,
	client.StatusStudy, client.StatusStayUp, client.StatusPlayBall, client.StatusSignal, client.StatusStudyOnline,
	client.StatusGaming, client.StatusVacationing, client.StatusWatchingTV, client.StatusFitness,
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

func newClient() *client.QQClient {
	c := client.NewClientEmpty()
	c.OnServerUpdated(func(bot *client.QQClient, e *client.ServerUpdatedEvent) bool {
		if !base.UseSSOAddress {
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
		case "DUMP":
			if !global.PathExists(global.DumpsPath) {
				_ = os.MkdirAll(global.DumpsPath, 0o755)
			}
			dumpFile := path.Join(global.DumpsPath, fmt.Sprintf("%v.dump", time.Now().Unix()))
			log.Errorf("出现错误 %v. 详细信息已转储至文件 %v 请连同日志提交给开发者处理", e.Message, dumpFile)
			_ = os.WriteFile(dumpFile, e.Dump, 0o644)
		}
	})
	return c
}

func (l *Dologin) saveToken() {
	base.AccountToken = l.Cli.GenToken()
	_ = os.WriteFile("session.token", base.AccountToken, 0o644)
}

// 退出程序,用于docker重启
func (l *Dologin) Shutdown(ctx iris.Context) {
	err := l.checkAuth(ctx)
	if err != nil {
		return
	}
	os.Exit(1)
}

func (l *Dologin) initLog() {
	base.SetConf(l.Config)
	rotateOptions := []rotatelogs.Option{
		rotatelogs.WithRotationTime(time.Hour * 24),
	}
	rotateOptions = append(rotateOptions, rotatelogs.WithMaxAge(base.LogAging))
	if base.LogForceNew {
		rotateOptions = append(rotateOptions, rotatelogs.ForceNewFile())
	}
	w, err := rotatelogs.New(path.Join("logs", "%Y-%m-%d.log"), rotateOptions...)
	if err != nil {
		log.Errorf("rotatelogs init err: %v", err)
		panic(err)
	}

	consoleFormatter := global.LogFormat{EnableColor: base.LogColorful}
	fileFormatter := global.LogFormat{EnableColor: false}
	log.AddHook(global.NewLocalHook(w, consoleFormatter, fileFormatter, global.GetLogLevel(base.LogLevel)...))

	mkCacheDir := func(path string, _type string) (errmsg string) {
		if !global.PathExists(path) {
			if err := os.MkdirAll(path, 0o755); err != nil {
				//log.Fatalf("创建%s缓存文件夹失败: %v", _type, err)
				return fmt.Sprintf("创建%s缓存文件夹失败: %v", _type, err)
			}
		}
		return ""
	}

	errmsg := mkCacheDir(global.ImagePath, "图片")
	errmsg += mkCacheDir(global.VoicePath, "语音")
	errmsg += mkCacheDir(global.VideoPath, "视频")
	errmsg += mkCacheDir(global.CachePath, "发送图片")
	errmsg += mkCacheDir(path.Join(global.ImagePath, "guild-images"), "频道图片缓存")
	if errmsg != ""{
		panic("配置信息有误")
	}
	cache.Init()
	db.Init()
	err = db.Open()
	if err != nil {
		log.Errorf("leveldb err:%v", err)
	}
}
