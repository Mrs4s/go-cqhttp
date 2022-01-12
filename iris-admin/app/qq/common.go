// Package gocq 程序的主体部分
package qq

import (
	"crypto/aes"
	"crypto/sha1"
	"encoding/hex"
	"github.com/Mrs4s/MiraiGo/client"
	"github.com/Mrs4s/go-cqhttp/global"
	"github.com/Mrs4s/go-cqhttp/internal/base"
	"github.com/kataras/iris/v12"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/pbkdf2"
	"os"
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
