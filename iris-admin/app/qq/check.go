package qq

import (
	"crypto/md5"
	"fmt"
	"github.com/GoAdminGroup/go-admin/engine"
	models2 "github.com/GoAdminGroup/go-admin/plugins/admin/models"
	"github.com/Mrs4s/MiraiGo/binary"
	"github.com/Mrs4s/MiraiGo/client"
	"github.com/Mrs4s/go-cqhttp/db"
	"github.com/Mrs4s/go-cqhttp/global"
	"github.com/Mrs4s/go-cqhttp/internal/base"
	"github.com/Mrs4s/go-cqhttp/internal/cache"
	"github.com/Mrs4s/go-cqhttp/iris-admin/models"
	"github.com/Mrs4s/go-cqhttp/iris-admin/utils/common"
	"github.com/Mrs4s/go-cqhttp/iris-admin/utils/jump"
	"github.com/kataras/iris/v12"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"os"
	"path"
	"sync"
	"time"
)

//验证 QQ是否已经登录
func (l *Dologin) CheckQQlogin(ctx iris.Context) error {
	err := l.checkAuth(ctx)
	if err != nil {
		return err
	}
	if !l.Cli.Online.Load() {
		jump.JumpErrorForIris(ctx, common.Msg{
			Msg:  "qq尚未登录",
			Url:  "/admin/qq/info",
			Wait: 3,
		})
		return errors.New("qq尚未登录")
	}
	return nil
}

func (l *Dologin) checkAuth(ctx iris.Context) error {
	var user models2.UserModel
	var ok bool
	user, ok = engine.User(ctx)
	if !ok {
		jump.JumpErrorForIris(ctx, common.Msg{
			Msg:  "获取登录信息失败",
			Url:  "/admin/",
			Wait: 3,
		})
		return errors.New("获取登录信息失败")
	}
	isAdmin := false
	for _, v := range user.Roles {
		if v.Id == 1 {
			isAdmin = true
		}
	}
	if !isAdmin {
		jump.JumpErrorForIris(ctx, common.Msg{
			Msg:  "非超级管理员",
			Url:  "/admin/",
			Wait: 3,
		})
		return errors.New("非超级管理员")
	}
	return nil
}

// 校验配置 分发登录地址
func (l *Dologin) CheckConfig(ctx iris.Context) {
	err := l.checkAuth(ctx)
	if err != nil {
		return
	}
	cfg, err := models.GetQqConfig()
	if err != nil {
		jump.JumpErrorForIris(ctx, common.Msg{
			Msg:  err.Error(),
			Url:  "/admin/info/qq_config",
			Wait: 3,
		})
		return
	}
	if l.Cli != nil && l.Cli.Online.Load() {
		jump.JumpSuccessForIris(ctx, common.Msg{
			Msg: "QQ已经在线",
			Url: "/admin/qq/info",
		})
		return
	}
	l.Config = cfg
	base.SetConf(l.Config)
	l.IsQRLogin = (base.Account.Uin == 0 || len(base.Account.Password) == 0) && !base.Account.Encrypt
	isTokenLogin := false
	var byteKey []byte
	byteKey, err = models.Getbytekey()
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
	if errmsg != "" {
		jump.JumpErrorForIris(ctx, common.Msg{
			Msg:  errmsg,
			Url:  "/admin/info/qq_config", //配置信息有误
			Wait: 3,
		})
		return
	}

	cache.Init()
	db.Init()
	db.Open()

	log.Info("当前版本:", base.Version)
	if base.Debug {
		log.SetLevel(log.DebugLevel)
		log.SetReportCaller(true)
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
	if global.PathExists("session.token") {
		token, err := os.ReadFile("session.token")
		if err == nil {
			if base.Account.Uin != 0 {
				r := binary.NewReader(token)
				cu := r.ReadInt64()
				if cu != base.Account.Uin {
					msg := fmt.Sprintf("警告: 配置文件内的QQ号 (%v) 与缓存内的QQ号 (%v) 不相同,已删除缓存，请重新登录", base.Account.Uin, cu)
					jump.JumpErrorForIris(ctx, common.Msg{
						Msg:  msg,
						Url:  "/admin/info/qq_config",
						Wait: 3,
					})
					return
				}
			}
			if err = l.Cli.TokenLogin(token); err != nil {
				_ = os.Remove("session.token")
				log.Warnf("恢复会话失败: %v , 尝试使用正常流程登录.", err)
				time.Sleep(time.Second)
				l.Cli.Disconnect()
				l.Cli.Release()
				l.Cli = newClient()
			} else {
				isTokenLogin = true
			}
		}
	}
	if base.Account.Uin != 0 && base.PasswordHash != [16]byte{} {
		l.Cli.Uin = base.Account.Uin
		l.Cli.PasswordMd5 = base.PasswordHash
	}
	if base.Account.Encrypt {
		if !global.PathExists("password.encrypt") {
			if base.Account.Password == "" {
				jump.JumpErrorForIris(ctx, common.Msg{
					Msg:  "无法进行加密，请在配置文件中的添加密码后重新启动.",
					Url:  "/admin/info/qq_config",
					Wait: 3,
				})
				return
			}

			if len(byteKey) == 0 {
				jump.JumpErrorForIris(ctx, common.Msg{
					Msg:  "密码加密已启用, 请输入Key对密码进行加密",
					Url:  "/admin/qq/encrypt_key_input",
					Wait: 3,
				})
				return
			}
		} else {
			if base.Account.Password != "" {
				jump.JumpErrorForIris(ctx, common.Msg{
					Msg:  "密码已加密，为了您的账号安全，请删除配置文件中的密码后重新启动.",
					Url:  "/admin/info/qq_config",
					Wait: 3,
				})
				return
			}
			if len(byteKey) == 0 {
				jump.JumpErrorForIris(ctx, common.Msg{
					Msg:  "密码加密已启用, 请输入Key对密码进行加密",
					Url:  "/admin/qq/encrypt_key_input",
					Wait: 3,
				})
				return
			}

			encrypt, _ := os.ReadFile("password.encrypt")
			ph, err := PasswordHashDecrypt(string(encrypt), byteKey)
			if err != nil {
				//log.Fatalf("加密存储的密码损坏，请尝试重新配置密码")
				jump.JumpErrorForIris(ctx, common.Msg{
					Msg:  "加密存储的密码损坏，请尝试重新配置密码",
					Url:  "/admin/info/qq_config",
					Wait: 3,
				})
				return
			}
			copy(base.PasswordHash[:], ph)
		}
	} else if len(base.Account.Password) > 0 {
		base.PasswordHash = md5.Sum([]byte(base.Account.Password))
	}
	l.Cli = newClient()
	var times uint = 1 // 重试次数
	var reLoginLock sync.Mutex
	l.Cli.OnDisconnected(func(q *client.QQClient, e *client.ClientDisconnectedEvent) {
		reLoginLock.Lock()
		defer reLoginLock.Unlock()
		times = 1
		if l.Cli.Online.Load() {
			return
		}
		log.Warnf("Bot已离线: %v", e.Message)
		time.Sleep(time.Second * time.Duration(base.Reconnect.Delay))
		for {
			if base.Reconnect.Disabled {
				log.Warnf("未启用自动重连, 将退出.")
				time.Sleep(time.Second)
				l.Cli.Disconnect()
				l.Cli.Release()
				l.Cli = newClient()
				l.ErrMsg = struct {
					Code int
					Msg  string
					Step int
				}{Code: 1000, Msg: "未启用自动重连, 将退出", Step: 1}
				return
			}
			if times > base.Reconnect.MaxTimes && base.Reconnect.MaxTimes != 0 {
				//log.Fatalf("Bot重连次数超过限制, 停止")
				time.Sleep(time.Second)
				l.Cli.Disconnect()
				l.Cli.Release()
				l.Cli = newClient()
				l.ErrMsg = struct {
					Code int
					Msg  string
					Step int
				}{Code: 1001, Msg: "Bot重连次数超过限制, 停止", Step: 1}
				return
			}
			times++
			if base.Reconnect.Interval > 0 {
				log.Warnf("将在 %v 秒后尝试重连. 重连次数：%v/%v", base.Reconnect.Interval, times, base.Reconnect.MaxTimes)
				time.Sleep(time.Second * time.Duration(base.Reconnect.Interval))
			} else {
				time.Sleep(time.Second)
			}
			if l.Cli.Online.Load() {
				log.Infof("登录已完成")
				break
			}
			log.Warnf("尝试重连...")
			err := l.Cli.TokenLogin(base.AccountToken)
			if err == nil {
				l.saveToken()
				return
			}
			log.Warnf("快速重连失败: %v", err)
			if l.IsQRLogin {
				//log.Fatalf("快速重连失败, 扫码登录无法恢复会话.")
				time.Sleep(time.Second)
				l.Cli.Disconnect()
				l.Cli.Release()
				l.Cli = newClient()
				l.ErrMsg = struct {
					Code int
					Msg  string
					Step int
				}{Code: 1002, Msg: "快速重连失败, 扫码登录无法恢复会话.", Step: 1}
				//panic("快速重连失败, 扫码登录无法恢复会话.")
				return
			}
			log.Warnf("快速重连失败, 尝试普通登录. 这可能是因为其他端强行T下线导致的.")
			time.Sleep(time.Second)
			if err := l.commonLogin(ctx); err != nil {
				//log.Errorf("登录时发生致命错误: %v", err)
				time.Sleep(time.Second)
				l.Cli.Disconnect()
				l.Cli.Release()
				l.Cli = newClient()
				l.ErrMsg = struct {
					Code int
					Msg  string
					Step int
				}{Code: 1002, Msg: fmt.Sprintf("登录时发生致命错误: %v", err), Step: 1}
				return
			} else {
				l.saveToken()
				break
			}
		}
	})
	if !isTokenLogin {
		if !l.IsQRLogin {
			if err := l.commonLogin(ctx); err != nil {
				log.Errorf("登录时发生致命错误： %v", err)
				return
			}
		} else {
			jump.JumpSuccessForIris(ctx, common.Msg{
				Msg: "将采用扫码登录",
				Url: "/qq/qrlogin",
			})
			return
			//ctx.Redirect("/qq/qrlogin", 302)
		}
	} else {
		jump.JumpSuccessForIris(ctx, common.Msg{
			Msg:  "自动登录成功",
			Url:  "/qq/loginsuccess", //普通方式登录地址
			Wait: 3,
		})
		return
	}
	if (base.Account.Uin == 0 || (base.Account.Password == "" && !base.Account.Encrypt)) && !global.PathExists("session.token") {
		msg := "账号密码未配置, 将使用二维码登录."
		var wait int64 = 3
		if !base.FastStart {
			msg += "将在 5秒 后继续."
			wait = 5
		}
		jump.JumpSuccessForIris(ctx, common.Msg{
			Msg:  msg,
			Url:  "/qq/qrlogin", //二维码方式登录地址
			Wait: wait,
		})
		return
	}

	jump.JumpSuccessForIris(ctx, common.Msg{
		Msg:  "配置校验通过，开始执行登录流程",
		Url:  "/qq/login", //普通方式登录地址
		Wait: 3,
	})
}
