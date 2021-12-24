package gocq

import (
	"crypto/md5"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/term"

	"github.com/Mrs4s/go-cqhttp/global"
	"github.com/Mrs4s/go-cqhttp/internal/base"
)

// CheckKey 检查密码
func CheckKey(byteKey []byte) {
	if base.Account.Encrypt {
		if !global.PathExists("password.encrypt") {
			if base.Account.Password == "" {
				log.Error("无法进行加密，请在配置文件中的添加密码后重新启动.")
				readLine()
				os.Exit(0)
			}
			log.Infof("密码加密已启用, 请输入Key对密码进行加密: (Enter 提交)")
			byteKey, _ = term.ReadPassword(int(os.Stdin.Fd()))
			base.PasswordHash = md5.Sum([]byte(base.Account.Password))
			_ = os.WriteFile("password.encrypt", []byte(PasswordHashEncrypt(base.PasswordHash[:], byteKey)), 0o644)
			log.Info("密码已加密，为了您的账号安全，请删除配置文件中的密码后重新启动.")
			readLine()
			os.Exit(0)
		} else {
			if base.Account.Password != "" {
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
	} else if len(base.Account.Password) > 0 {
		base.PasswordHash = md5.Sum([]byte(base.Account.Password))
	}
}
