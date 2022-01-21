package qq

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"github.com/GoAdminGroup/go-admin/context"
	"github.com/GoAdminGroup/go-admin/modules/auth"
	"github.com/GoAdminGroup/go-admin/modules/config"
	"github.com/GoAdminGroup/go-admin/modules/db"
	tmpl "github.com/GoAdminGroup/go-admin/template"
	"github.com/GoAdminGroup/go-admin/template/icon"
	"github.com/GoAdminGroup/go-admin/template/types"
	"github.com/GoAdminGroup/go-admin/template/types/form"
	"github.com/Mrs4s/MiraiGo/binary"
	"github.com/Mrs4s/MiraiGo/client"
	"github.com/Mrs4s/go-cqhttp/coolq"
	"github.com/Mrs4s/go-cqhttp/global"
	"github.com/Mrs4s/go-cqhttp/internal/base"
	"github.com/Mrs4s/go-cqhttp/iris-admin/loghook"
	"github.com/Mrs4s/go-cqhttp/iris-admin/models"
	"github.com/Mrs4s/go-cqhttp/iris-admin/utils/common"
	"github.com/Mrs4s/go-cqhttp/iris-admin/utils/jump"
	config2 "github.com/Mrs4s/go-cqhttp/modules/config"
	"github.com/Mrs4s/go-cqhttp/modules/servers"
	"github.com/kataras/iris/v12"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/writer"
	"os"
	"strings"
	"sync"
	"time"
)

// web上操作登录qq

type Dologin struct {
	Cli       *client.QQClient            //qqcli
	Qrcode    *client.QRCodeLoginResponse //当前的二维码
	IsQRLogin bool
	Captcha   *client.QRCodeLoginResponse //当前的验证码
	Config    *config2.Config
	// 错误信息 异步获取
	ErrMsg struct {
		Code int
		Msg  string
		Step int
	}
	Status bool //是否已经启动过net server
	Conn   chan string
	Bot    *coolq.CQBot
	Weblog *loghook.WebLogWriter
}

// 初始化
func NewDologin() *Dologin {
	cfg, _ := models.GetQqConfig()
	weblog := loghook.NewWebLogWriter()
	log.AddHook(&writer.Hook{
		Writer: weblog,
		LogLevels: []log.Level{
			log.PanicLevel,
			log.FatalLevel,
			log.ErrorLevel,
			log.WarnLevel,
			log.InfoLevel,
			log.DebugLevel,
		},
	})
	return &Dologin{
		Cli:    newClient(),
		Config: cfg,
		Conn:   make(chan string, 255),
		ErrMsg: struct {
			Code int
			Msg  string
			Step int
		}{Code: 0, Msg: "", Step: 0},
		Status: false,
		Weblog: weblog,
	}
}

// 密码加密的页面
func (l *Dologin) EncryptPasswordEnterWeb(ctx *context.Context) (types.Panel, error) {
	// 获取登录用户模型
	user := auth.Auth(ctx)
	log.Debugf("user = %v", user)
	components := tmpl.Get(config.GetTheme())

	btn1 := components.Button().SetType("submit").
		SetContent("确认提交").
		SetThemePrimary().
		SetOrientationRight().
		SetLoadingText(icon.Icon("fa-spinner fa-spin", 2) + `Save`).
		GetContent()
	link := tmpl.Default().Link().
		SetURL("/admin/info/qq_config"). // 设置跳转路由
		SetContent("返回配置页面"). // 设置链接内容
		SetClass("btn-group btn btn-sm btn-info btn-flat pull-left").
		GetContent()
	col1 := components.Col().SetSize(types.SizeMD(8)).SetContent(link).GetContent()
	btn2 := components.Button().SetType("reset").
		SetContent("重置").
		SetThemeWarning().
		SetOrientationLeft().
		GetContent()
	col2 := components.Col().SetSize(types.SizeMD(8)).
		SetContent(btn1 + btn2).GetContent()
	var panel = types.NewFormPanel()

	panel.AddField("密码加密key", "byteKey", db.Varchar, form.Text).FieldPlaceholder("密码加密和解密都需要的key").FieldMust()
	panel.SetTabGroups(types.TabGroups{
		{"byteKey"},
	})
	panel.SetTabHeaders("你开启了密码加密功能，请输入你的加密算法的key")
	fields, headers := panel.GroupField()
	aform := components.Form().
		SetTabHeaders(headers).
		SetTabContents(fields).
		SetPrefix(config.PrefixFixSlash()).
		SetUrl("/qq/do_encrypt_key_input").
		//SetTitle("Form").
		SetOperationFooter(col1 + col2)
	//SetOperationFooter(`<a href="/admin/info/qq_config"   class="btn btn-sm btn-info btn-flat pull-left">返回配置页面</a><a href="javascript:void(0)" id="SubmitPWDKEY" class="btn btn-sm btn-default btn-flat pull-right">确认提交</a>`)

	return types.Panel{
		Content: components.Box().
			SetHeader(aform.GetDefaultBoxHeader(true)).
			WithHeadBorder().
			SetBody(aform.GetContent()).
			GetContent(),
		Title:       "密码加密key输入页面",
		Callbacks:   panel.Callbacks,
		Description: "你选择了加密密码，请输入加密算法的key",
	}, nil
}

// bytekey的输入处理
func (l *Dologin) DoEncryptKeyInput(ctx iris.Context) (types.Panel, error) {
	byteKey := ctx.FormValue("byteKey")
	if byteKey == "" {
		return jump.JumpError(common.Msg{
			Msg:  "输入的bytekey为空，请重新输入",
			Url:  "/admin/qq/encrypt_key_input", //二维码方式登录地址
			Wait: 3,
		}), nil
	}
	base.PasswordHash = md5.Sum([]byte(base.Account.Password))
	_ = os.WriteFile("password.encrypt", []byte(PasswordHashEncrypt(base.PasswordHash[:], []byte(byteKey))), 0o644)
	log.Info("密码已加密，为了您的账号安全，请删除配置文件中的密码后重新启动.")
	models.Setbytekey(byteKey)
	return jump.JumpSuccess(common.Msg{
		Msg:  "密码已加密，为了您的账号安全，请删除配置文件中的密码后重新启动.",
		Url:  "/admin/info/qq_config",
		Wait: 3,
	}), nil
}

// html session_token 选择页面
func (l *Dologin) SessionTokenWeb(ctx *context.Context) (types.Panel, error) {
	// 获取登录用户模型
	user := auth.Auth(ctx)
	log.Debugf("user = %v", user)
	components := tmpl.Get(config.GetTheme())

	btn1 := components.Button().SetType("submit").
		SetContent("确认提交").
		SetThemePrimary().
		SetOrientationRight().
		SetLoadingText(icon.Icon("fa-spinner fa-spin", 2) + `Save`).
		GetContent()
	link := tmpl.Default().Link().
		SetURL("/admin/info/qq_config"). // 设置跳转路由
		SetContent("返回配置页面"). // 设置链接内容
		//OpenInNewTab().  // 是否在新的tab页打开
		//SetTabTitle("Manager Detail").  // 设置tab的标题
		GetContent()
	col1 := components.Col().SetSize(types.SizeMD(8)).SetContent(link).GetContent()
	btn2 := components.Button().SetType("reset").
		SetContent("重置").
		SetThemeWarning().
		SetOrientationLeft().
		GetContent()
	col2 := components.Col().SetSize(types.SizeMD(8)).
		SetContent(btn1 + btn2).GetContent()
	var panel = types.NewFormPanel()

	panel.AddField("编号选择", "num", db.Varchar, form.SelectSingle).
		FieldOptions(types.FieldOptions{
			{Text: "使用会话缓存继续", Value: "1"},
			{Text: "删除会话缓存并重启.", Value: "2"},
		}).FieldDefault("1")
	panel.SetTabGroups(types.TabGroups{
		{"num"},
	})
	panel.SetTabHeaders("警告: 配置文件内的QQ号 与缓存内的QQ号  不相同")
	fields, headers := panel.GroupField()
	aform := components.Form().
		SetTabHeaders(headers).
		SetTabContents(fields).
		SetPrefix(config.PrefixFixSlash()).
		SetUrl("/qq/do_session_select").
		//SetTitle("Form").
		SetOperationFooter(col1 + col2)

	return types.Panel{
		Content: components.Box().
			SetHeader(aform.GetDefaultBoxHeader(true)).
			WithHeadBorder().
			SetBody(aform.GetContent()).
			GetContent(),
		//Title:       "密码加密key输入页面",
		Callbacks: panel.Callbacks,
		//Description: "你选择了加密密码，请输入加密算法的key",
	}, nil
}

func (l *Dologin) DoSessionTokenSelect(ctx iris.Context) (types.Panel, error) {
	num := ctx.FormValue("num")
	switch num {
	case "1":

	case "2":
	default:
		return jump.JumpError(common.Msg{
			Msg:  "输入的bytekey为空，请重新输入",
			Url:  "/admin/qq/encrypt_key_input", //二维码方式登录地址
			Wait: 3,
		}), nil
	}
	return types.Panel{}, nil
}

// 获取 二维码
func (l *Dologin) QrloginHtml(ctx iris.Context) {
	err := l.checkAuth(ctx)
	if err != nil {
		return
	}
	l.Qrcode, err = l.Cli.FetchQRCode()
	if err != nil {
		jump.JumpErrorForIris(ctx, common.Msg{
			Msg:  err.Error(),
			Url:  "/admin/info/qq_config",
			Wait: 3,
		})
	}
	ctx.Redirect("/qq/do_qrlogin", 302)
}

// 二维码登录处理
func (l *Dologin) DoQrlogin(ctx iris.Context) (types.Panel, error) {
	err := l.checkAuth(ctx)
	if err != nil {
		return types.Panel{}, nil
	}
	// 处理 input表单提交的数据
	if ctx.Method() == "POST" {
		var (
			res *client.LoginResponse
			err error
		)
		switch ctx.FormValue("action") {
		case "enterSMS":
			text := ctx.FormValue("smstext")
			res, err = l.Cli.SubmitSMS(text)
		case "captcha":
			text := ctx.FormValue("captchatext")
			res, err = l.Cli.SubmitCaptcha(text, l.Captcha.Sig)
		}
		err = l.loginResponseProcessor(ctx, res)
		if err != nil {
			return types.Panel{}, nil
		}
	}
	var header string
	if l.Cli.Uin != 0 {
		log.Infof("请使用账号 %v 登录手机QQ扫描二维码 (qrcode.png) : ", l.Cli.Uin)
		header = fmt.Sprintf("请使用账号 %v 登录手机QQ扫描二维码 (qrcode.png) : ", l.Cli.Uin)
	} else {
		log.Infof("请使用手机QQ扫描二维码 (qrcode.png) : ")
		header = fmt.Sprintf("请使用手机QQ扫描二维码 (qrcode.png) : ")
	}
	if l.Qrcode == nil {
		l.fetchQrCode()
	}
	s, err := l.Cli.QueryQRCodeStatus(l.Qrcode.Sig)
	if err != nil {
		l.fetchQrCode()
	}
	imgsrcBase64 := fmt.Sprintf("data:image/png;base64,%s", base64.StdEncoding.EncodeToString(l.Qrcode.ImageData))
	image := tmpl.Default().Image().SetSrc(tmpl.HTML(imgsrcBase64)).SetHeight("400").SetWidth("400").GetContent()
	var msg string
	time.Sleep(time.Second)
	s, _ = l.Cli.QueryQRCodeStatus(l.Qrcode.Sig)
	switch s.State {
	case client.QRCodeCanceled:
		//log.Fatalf("扫码被用户取消.")
		msg = "扫码被用户取消."
		l.fetchQrCode()
	case client.QRCodeTimeout:
		//log.Fatalf("二维码过期")
		msg = "二维码过期"
		l.fetchQrCode()
	case client.QRCodeWaitingForConfirm:
		//log.Infof("扫码成功, 请在手机端确认登录.")
		msg = "扫码成功, 请在手机端确认登录."
	case client.QRCodeConfirmed:
		res, err := l.Cli.QRCodeLogin(s.LoginInfo)
		if err != nil {
			jump.JumpErrorForIris(ctx, common.Msg{
				Msg:  err.Error(),
				Url:  "/admin/info/qq_config",
				Wait: 2,
			})
			return types.Panel{}, nil
		}
		err = l.loginResponseProcessor(ctx, res)
		if err != nil {
			return types.Panel{}, nil
		}
	case client.QRCodeImageFetch, client.QRCodeWaitingForScan:
		// ignore
	}
	//info:
	log.Info("输出页面")

	label := tmpl.Default().Label().
		SetContent(tmpl.HTML(msg)).
		SetType("info").GetContent()
	box := tmpl.Default().Box().
		WithHeadBorder().
		SetHeader("QQ扫码登录").
		SetBody(label + image).
		GetContent()
	return types.Panel{
		Title:           "QQ扫码登录",
		Description:     tmpl.HTML(header),
		Content:         box,
		AutoRefresh:     true,
		RefreshInterval: []int{1},
	}, nil
}

func (l *Dologin) loginResponseProcessor(ctx iris.Context, res *client.LoginResponse) error {
	var err error
	if err != nil {
		jump.JumpErrorForIris(ctx, common.Msg{
			Msg:  err.Error(),
			Url:  "/admin/info/qq_config",
			Wait: 2,
		})
		return err
	}
	if res.Success {
		jump.JumpSuccessForIris(ctx, common.Msg{
			Msg: "登录成功",
			Url: "/qq/loginsuccess",
		})
		return errors.New("登录成功")
	}
	time.Sleep(time.Second)
	switch res.Error {
	case client.SliderNeededError:
		log.Warnf("登录需要滑条验证码, 请使用手机QQ扫描二维码以继续登录.")
		l.Cli.Disconnect()
		l.Cli.Release()
		l.Cli = client.NewClientEmpty()
		jump.JumpErrorForIris(ctx, common.Msg{
			Msg: "登录需要滑条验证码, 请使用手机QQ扫描二维码以继续登录.",
			Url: "/qq/qrlogin",
		})
		return errors.New("登录需要滑条验证码, 请使用手机QQ扫描二维码以继续登录.")
	case client.NeedCaptcha:
		log.Warnf("登录需要验证码.")
		//_ = os.WriteFile("captcha.jpg", res.CaptchaImage, 0o644)
		//log.Warnf("请输入验证码 (captcha.jpg)： (Enter 提交)")
		jump.JumpErrorForIris(ctx, common.Msg{
			Msg: "登录需要验证码",
			Url: "/qq/captcha_input",
		})
		return errors.New("登录需要验证码.")
	case client.SMSNeededError:
		log.Warnf("账号已开启设备锁,  向手机 %v 发送短信验证码.", res.SMSPhone)
		if !l.Cli.RequestSMS() {
			log.Warnf("发送验证码失败，可能是请求过于频繁.")
			jump.JumpErrorForIris(ctx, common.Msg{
				Msg: fmt.Sprintf("账号已开启设备锁,  向手机 %v 发送短信验证码.", res.SMSPhone) + "发送验证码失败，可能是请求过于频繁.",
				Url: "/qq/qrlogin",
			})
			return errors.New("发送验证码失败，可能是请求过于频繁.")
		}
		jump.JumpErrorForIris(ctx, common.Msg{
			Msg: fmt.Sprintf("账号已开启设备锁,  向手机 %v 发送短信验证码.", res.SMSPhone),
			Url: "/qq/sms_input",
		})
		return errors.Errorf("账号已开启设备锁,  向手机 %v 发送短信验证码.", res.SMSPhone)
	case client.SMSOrVerifyNeededError:
		//log.Warnf("账号已开启设备锁，请选择验证方式:")
		//log.Warnf("1. 向手机 %v 发送短信验证码", res.SMSPhone)
		//log.Warnf("2. 使用手机QQ扫码验证.")
		//log.Warn("请输入(1 - 2) (将在10秒后自动选择2)：")
		//text = readLineTimeout(time.Second*10, "2")
		//if strings.Contains(text, "1") {
		//	if !l.Cli.RequestSMS() {
		//		log.Warnf("发送验证码失败，可能是请求过于频繁.")
		//		return errors.WithStack(ErrSMSRequestError)
		//	}
		//	log.Warn("请输入短信验证码： (Enter 提交)")
		//	text = readLine()
		//	res, err = l.Cli.SubmitSMS(text)
		//	continue
		//}
		//fallthrough
		if !l.Cli.RequestSMS() {
			log.Warnf("发送验证码失败，可能是请求过于频繁.")
			jump.JumpErrorForIris(ctx, common.Msg{
				Msg: fmt.Sprintf("账号已开启设备锁,  向手机 %v 发送短信验证码.", res.SMSPhone) + "发送验证码失败，可能是请求过于频繁.",
				Url: "/qq/qrlogin",
			})
			return errors.New("发送验证码失败，可能是请求过于频繁.")
		}
		jump.JumpErrorForIris(ctx, common.Msg{
			Msg: fmt.Sprintf("账号已开启设备锁,  向手机 %v 发送短信验证码.", res.SMSPhone),
			Url: "/qq/sms_input",
		})
		return errors.Errorf("账号已开启设备锁,  向手机 %v 发送短信验证码.", res.SMSPhone)
	case client.UnsafeDeviceError:
		//log.Warnf("账号已开启设备锁，请前往 -> %v <- 验证后重启Bot.", res.VerifyUrl)
		//log.Infof("按 Enter 或等待 5s 后继续....")
		//readLineTimeout(time.Second*5, "")
		//os.Exit(0)
		jump.JumpErrorForIris(ctx, common.Msg{
			Msg: fmt.Sprintf("账号已开启设备锁，请前往 -> %v <- 验证后重启Bot.", res.VerifyUrl),
			Url: res.VerifyUrl,
		})
		return errors.Errorf("账号已开启设备锁，请前往 -> %v <- 验证后重启Bot.", res.VerifyUrl)
	case client.OtherLoginError, client.UnknownLoginError, client.TooManySMSRequestError:
		msg := res.ErrorMessage
		if strings.Contains(msg, "版本") {
			msg = "密码错误或账号被冻结"
			jump.JumpErrorForIris(ctx, common.Msg{
				Msg: msg,
				Url: "/admin/info/qq_config",
			})
			return errors.New("密码错误或账号被冻结")
		}
		if strings.Contains(msg, "冻结") {
			jump.JumpErrorForIris(ctx, common.Msg{
				Msg: "账号被冻结",
				Url: "/admin/info/qq_config",
			})
			return errors.New("密码错误或账号被冻结")
			//log.Fatalf("账号被冻结")
		}
		log.Warnf("登录失败: %v", msg)
		jump.JumpErrorForIris(ctx, common.Msg{
			Msg: fmt.Sprintf("登录失败: %v", msg),
			Url: "/admin/info/qq_config",
		})
		return errors.Errorf("登录失败: %v", msg)
	}
	return nil
}

// 图片验证码的html输入页面
func (l *Dologin) CaptchaInput(ctx iris.Context) (types.Panel, error) {
	imgsrcBase64 := fmt.Sprintf("data:image/png;base64,%s", base64.StdEncoding.EncodeToString(l.Captcha.ImageData))
	image := tmpl.Default().Image().SetSrc(tmpl.HTML(imgsrcBase64)).GetContent()
	log.Info("输出页面")

	var panel = types.NewFormPanel()
	panel.AddField("图片验证码", "captchatext", db.Varchar, form.Text).FieldMust().FieldPlaceholder("输入验证码")
	panel.SetTabGroups(types.TabGroups{
		{"captchatext"},
	})
	fields, headers := panel.GroupField()
	components := tmpl.Get(config.GetTheme())
	panel.SetTabHeaders("图片验证码输入")
	btn1 := components.Button().SetType("submit").
		SetContent("确认提交").
		SetThemePrimary().
		SetOrientationRight().
		SetLoadingText(icon.Icon("fa-spinner fa-spin", 2) + `Save`).
		GetContent()
	btn2 := components.Button().SetType("reset").
		SetContent("重置").
		SetThemeWarning().
		SetOrientationLeft().
		GetContent()
	col2 := components.Col().SetSize(types.SizeMD(8)).
		SetContent(btn1 + btn2).GetContent()
	aform := tmpl.Default().Form().
		SetTabHeaders(headers).
		SetTabContents(fields).
		SetPrefix(config.PrefixFixSlash()).
		SetHiddenFields(map[string]string{
			"action": "captcha",
		}).
		SetUrl("/qq/do_qrlogin"). // 设置表单请求路由
		SetOperationFooter(col2).
		GetContent()
	box := tmpl.Default().Box().
		WithHeadBorder().
		SetHeader("QQ扫码登录").
		SetBody(aform + image).
		GetContent()
	return types.Panel{
		Title:       "验证码输入",
		Description: "登录需要验证码",
		Content:     box,
	}, nil
}

// 短信验证码的html输入页面
func (l *Dologin) SmsInput(ctx *context.Context) (types.Panel, error) {
	var panel = types.NewFormPanel()
	panel.AddField("短信验证码", "smstext", db.Varchar, form.Text).FieldMust().FieldPlaceholder("输入验证码")
	panel.SetTabGroups(types.TabGroups{
		{"smstext"},
	})
	panel.SetTabHeaders("短信验证码输入")
	fields, headers := panel.GroupField()
	components := tmpl.Get(config.GetTheme())

	btn1 := components.Button().SetType("submit").
		SetContent("确认提交").
		SetThemePrimary().
		SetOrientationRight().
		SetLoadingText(icon.Icon("fa-spinner fa-spin", 2) + `Save`).
		GetContent()
	btn2 := components.Button().SetType("reset").
		SetContent("重置").
		SetThemeWarning().
		SetOrientationLeft().
		GetContent()
	col2 := components.Col().SetSize(types.SizeMD(8)).
		SetContent(btn2 + btn1).GetContent()
	aform := tmpl.Default().Form().
		SetTabHeaders(headers).
		SetTabContents(fields).
		SetPrefix(config.PrefixFixSlash()).
		SetHiddenFields(map[string]string{
			"action": "enterSMS",
		}).
		SetUrl("/qq/do_qrlogin"). // 设置表单请求路由
		SetOperationFooter(col2).
		GetContent()
	box := tmpl.Default().Box().
		WithHeadBorder().
		SetBody(aform).
		SetHeader("短信验证码输入").
		GetContent()
	return types.Panel{
		Title:       "验证码输入",
		Description: "登录需要验证码",
		Content:     box,
	}, nil
}

func (l *Dologin) commonLogin(ctx iris.Context) error {
	res, err := l.Cli.Login()
	if err != nil {
		jump.JumpErrorForIris(ctx, common.Msg{
			Msg: fmt.Sprintf("common 登录失败:%s", err.Error()),
			Url: "/admin/info/qq_config",
		})
		return err
	}
	return l.loginResponseProcessor(ctx, res)
}

func (l *Dologin) fetchQrCode() {
	l.Qrcode, _ = l.Cli.FetchQRCode()
}

func (l *Dologin) DoLoginBackend() {
	for {
		select {
		case str := <-l.Conn:
			fmt.Println(str)
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
					res, err := l.Cli.Login()
					if err != nil {
						l.ErrMsg = struct {
							Code int
							Msg  string
							Step int
						}{Code: 3003, Msg: "登录失败：" + err.Error(), Step: 5}
						return
					}
					if err := l.loginResponseProcessorBackend(res); err != nil {
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
			l.saveToken()
			l.Cli.AllowSlider = true
			log.Infof("使用协议: %s", client.SystemDeviceInfo.Protocol)
			log.Infof("登录成功 欢迎使用: %v", l.Cli.Nickname)
			log.Info("开始加载好友列表...")
			global.Check(l.Cli.ReloadFriendList(), true)
			log.Infof("共加载 %v 个好友.", len(l.Cli.FriendList))
			log.Infof("开始加载群列表...")
			global.Check(l.Cli.ReloadGroupList(), true)
			log.Infof("共加载 %v 个群.", len(l.Cli.GroupList))
			if uint(base.Account.Status) >= uint(len(allowStatus)) {
				base.Account.Status = 0
			}
			l.Cli.SetOnlineStatus(allowStatus[base.Account.Status])
			if l.Status {
				l.Bot.SetClient(l.Cli)
			} else {
				l.Status = true
				l.Bot = coolq.NewQQBot(l.Cli)
				servers.Run(l.Bot)
			}
			l.ErrMsg = struct {
				Code int
				Msg  string
				Step int
			}{Code: 0, Msg: "登录成功", Step: 0}
			//servers.Run(coolq.NewQQBot(l.Cli))
			log.Info("资源初始化完成, 开始处理信息.")
			log.Info("アトリは、高性能ですから!")

		}
	}
}

func (l *Dologin) LoginSuccess(ctx iris.Context) {
	l.Conn <- "loginsuccess"
	ctx.Redirect("/admin/qq/info")
}

func (l *Dologin) NomalLogin(ctx iris.Context) {
	err := l.checkAuth(ctx)
	if err != nil {
		return
	}
	cfg, err := models.GetQqConfig()
	if err != nil {
		jump.JumpErrorForIris(ctx, common.Msg{
			Msg:  err.Error(),
			Url:  "/admin/info/qq_config",
			Wait: 2,
		})
		return
	}
	l.Config = cfg
	base.SetConf(cfg)
	if err := l.commonLogin(ctx); err != nil {
		return
	}
}

// 自动登录尝试
func (l *Dologin) AutoLoginCommon() {
	cfg, err := models.GetQqConfig()
	if err != nil {
		l.ErrMsg = struct {
			Code int
			Msg  string
			Step int
		}{Code: 3000, Msg: "配置信息获取错误", Step: 0}
		return
	}
	if l.Cli != nil && l.Cli.Online.Load() {
		l.ErrMsg = struct {
			Code int
			Msg  string
			Step int
		}{Code: 0, Msg: "QQ已经在线", Step: 0}
		return
	}
	l.Config = cfg
	base.SetConf(l.Config)
	l.IsQRLogin = (base.Account.Uin == 0 || len(base.Account.Password) == 0) && !base.Account.Encrypt
	isTokenLogin := false
	var byteKey []byte
	byteKey, err = models.Getbytekey()
	l.initLog()
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
	l.Cli = newClient()
	//var times uint = 1 // 重试次数
	//var reLoginLock sync.Mutex
	//l.Cli.OnDisconnected(func(q *client.QQClient, e *client.ClientDisconnectedEvent) {
	//	reLoginLock.Lock()
	//	defer reLoginLock.Unlock()
	//	times = 1
	//	if l.Cli.Online.Load() {
	//		return
	//	}
	//	log.Warnf("Bot已离线: %v", e.Message)
	//	time.Sleep(time.Second * time.Duration(base.Reconnect.Delay))
	//	for {
	//		if base.Reconnect.Disabled {
	//			log.Warnf("未启用自动重连, 将退出.")
	//			time.Sleep(time.Second)
	//			l.Cli.Disconnect()
	//			l.Cli.Release()
	//			l.Cli = newClient()
	//			l.ErrMsg = struct {
	//				Code int
	//				Msg  string
	//				Step int
	//			}{Code: 1000, Msg: "未启用自动重连, 将退出", Step: 1}
	//			return
	//		}
	//		if times > base.Reconnect.MaxTimes && base.Reconnect.MaxTimes != 0 {
	//			//log.Fatalf("Bot重连次数超过限制, 停止")
	//			time.Sleep(time.Second)
	//			l.Cli.Disconnect()
	//			l.Cli.Release()
	//			l.Cli = newClient()
	//			l.ErrMsg = struct {
	//				Code int
	//				Msg  string
	//				Step int
	//			}{Code: 1001, Msg: "Bot重连次数超过限制, 停止", Step: 1}
	//			return
	//		}
	//		times++
	//		if base.Reconnect.Interval > 0 {
	//			log.Warnf("将在 %v 秒后尝试重连. 重连次数：%v/%v", base.Reconnect.Interval, times, base.Reconnect.MaxTimes)
	//			time.Sleep(time.Second * time.Duration(base.Reconnect.Interval))
	//		} else {
	//			time.Sleep(time.Second)
	//		}
	//		if l.Cli.Online.Load() {
	//			log.Infof("登录已完成")
	//			break
	//		}
	//		log.Warnf("尝试重连...")
	//		err := l.Cli.TokenLogin(base.AccountToken)
	//		if err == nil {
	//			l.saveToken()
	//			return
	//		}
	//		log.Warnf("快速重连失败: %v", err)
	//		if l.IsQRLogin {
	//			//log.Fatalf("快速重连失败, 扫码登录无法恢复会话.")
	//			time.Sleep(time.Second)
	//			l.Cli.Disconnect()
	//			l.Cli.Release()
	//			l.Cli = newClient()
	//			l.ErrMsg = struct {
	//				Code int
	//				Msg  string
	//				Step int
	//			}{Code: 1002, Msg: "快速重连失败, 扫码登录无法恢复会话.", Step: 1}
	//			//panic("快速重连失败, 扫码登录无法恢复会话.")
	//			return
	//		}
	//		log.Warnf("快速重连失败, 尝试普通登录. 这可能是因为其他端强行T下线导致的.")
	//		time.Sleep(time.Second)
	//		res, err := l.Cli.Login()
	//		if err != nil {
	//			l.ErrMsg = struct {
	//				Code int
	//				Msg  string
	//				Step int
	//			}{Code: 3003, Msg: "登录失败：" + err.Error(), Step: 5}
	//			return
	//		}
	//		if err := l.loginResponseProcessorBackend(res); err != nil {
	//			//log.Errorf("登录时发生致命错误: %v", err)
	//			time.Sleep(time.Second)
	//			l.Cli.Disconnect()
	//			l.Cli.Release()
	//			l.Cli = newClient()
	//			l.ErrMsg = struct {
	//				Code int
	//				Msg  string
	//				Step int
	//			}{Code: 1002, Msg: fmt.Sprintf("登录时发生致命错误: %v", err), Step: 1}
	//			return
	//		} else {
	//			l.saveToken()
	//			break
	//		}
	//	}
	//})
	if global.PathExists("session.token") {
		token, err := os.ReadFile("session.token")
		if err == nil {
			if base.Account.Uin != 0 {
				r := binary.NewReader(token)
				cu := r.ReadInt64()
				if cu != base.Account.Uin {
					msg := fmt.Sprintf("警告: 配置文件内的QQ号 (%v) 与缓存内的QQ号 (%v) 不相同,已删除缓存，请重新登录", base.Account.Uin, cu)
					l.ErrMsg = struct {
						Code int
						Msg  string
						Step int
					}{Code: 3005, Msg: msg, Step: 0}
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
				l.ErrMsg = struct {
					Code int
					Msg  string
					Step int
				}{Code: 3006, Msg: "自动登录失败： 无法进行加密，请在配置文件中的添加密码后重新启动.", Step: 0}
				return
			}

			if len(byteKey) == 0 {
				l.ErrMsg = struct {
					Code int
					Msg  string
					Step int
				}{Code: 3006, Msg: "自动登录失败： 无法进行加密，请在配置文件中的添加密码后重新启动.", Step: 0}
				return
			}
		} else {
			if base.Account.Password != "" {
				l.ErrMsg = struct {
					Code int
					Msg  string
					Step int
				}{Code: 3006, Msg: "自动登录失败：密码已加密，为了您的账号安全，请删除配置文件中的密码后重新启动.", Step: 0}
				return
			}
			if len(byteKey) == 0 {
				l.ErrMsg = struct {
					Code int
					Msg  string
					Step int
				}{Code: 3006, Msg: "自动登录失败：无法进行加密，请在配置文件中的添加密码后重新启动.", Step: 0}
				return
			}

			encrypt, _ := os.ReadFile("password.encrypt")
			ph, err := PasswordHashDecrypt(string(encrypt), byteKey)
			if err != nil {
				l.ErrMsg = struct {
					Code int
					Msg  string
					Step int
				}{Code: 3006, Msg: "自动登录失败：加密存储的密码损坏，请尝试重新配置密码", Step: 0}
				return
			}
			copy(base.PasswordHash[:], ph)
		}
	} else if len(base.Account.Password) > 0 {
		base.PasswordHash = md5.Sum([]byte(base.Account.Password))
	}

	if !isTokenLogin {
		if !l.IsQRLogin {
			res, err := l.Cli.Login()
			if err != nil {
				l.ErrMsg = struct {
					Code int
					Msg  string
					Step int
				}{Code: 3003, Msg: "登录失败：" + err.Error(), Step: 5}
				return
			}
			if err := l.loginResponseProcessorBackend(res); err != nil {
				log.Errorf("登录时发生致命错误： %v", err)
				return
			}
		} else {
			l.ErrMsg = struct {
				Code int
				Msg  string
				Step int
			}{Code: 3007, Msg: "自动登录失败，需要扫码登录", Step: 1}
		}
	}
	if isTokenLogin {
		l.ErrMsg.Msg = "自动登录成功"
		l.Conn <- "loginsuccess"
		return
	}
	if (base.Account.Uin == 0 || (base.Account.Password == "" && !base.Account.Encrypt)) && !global.PathExists("session.token") {
		log.Warnf("账号密码未配置, 将使用二维码登录.")
		l.ErrMsg = struct {
			Code int
			Msg  string
			Step int
		}{Code: 2000, Msg: "账号密码未配置, 将使用二维码登录.", Step: 1}
		return
	}
}

// 启动的时候自动登录 回包处理
func (l *Dologin) loginResponseProcessorBackend(res *client.LoginResponse) error {
	var err error
	for {
		if err != nil {
			l.ErrMsg.Msg = err.Error()
			l.ErrMsg.Code = 2000
			return err
		}
		if res.Success {
			l.ErrMsg.Msg = "自动登录成功"
			l.Conn <- "loginsuccess"
			return nil
		}
		time.Sleep(time.Second)
		switch res.Error {
		case client.SliderNeededError:
			log.Warnf("登录需要滑条验证码, 请使用手机QQ扫描二维码以继续登录.")
			l.Cli.Disconnect()
			l.Cli.Release()
			l.Cli = client.NewClientEmpty()
			l.ErrMsg = struct {
				Code int
				Msg  string
				Step int
			}{Code: 2001, Msg: "登录需要滑条验证码, 请使用手机QQ扫描二维码以继续登录.", Step: 2}
			return errors.New("登录需要滑条验证码, 请使用手机QQ扫描二维码以继续登录.")
		case client.NeedCaptcha:
			log.Warnf("登录需要验证码.")
			//_ = os.WriteFile("captcha.jpg", res.CaptchaImage, 0o644)
			//log.Warnf("请输入验证码 (captcha.jpg)： (Enter 提交)")
			l.ErrMsg = struct {
				Code int
				Msg  string
				Step int
			}{Code: 2002, Msg: "自动登录失败，需要验证码", Step: 2}
			return errors.New("登录需要验证码.")
		case client.SMSNeededError:
			log.Warnf("账号已开启设备锁,  向手机 %v 发送短信验证码.", res.SMSPhone)
			if !l.Cli.RequestSMS() {
				log.Warnf("发送验证码失败，可能是请求过于频繁.")
				l.ErrMsg = struct {
					Code int
					Msg  string
					Step int
				}{Code: 2003, Msg: "自动登录失败，发送验证码失败，可能是请求过于频繁.", Step: 2}
				return errors.New("发送验证码失败，可能是请求过于频繁.")
			}
			l.ErrMsg = struct {
				Code int
				Msg  string
				Step int
			}{Code: 2004, Msg: "自动登录失败，账号已开启设备锁.需要验证码", Step: 2}
			return errors.Errorf("账号已开启设备锁,  向手机 %v 发送短信验证码.", res.SMSPhone)
		case client.SMSOrVerifyNeededError:
			if !l.Cli.RequestSMS() {
				log.Warnf("发送验证码失败，可能是请求过于频繁.")
				l.ErrMsg = struct {
					Code int
					Msg  string
					Step int
				}{Code: 2003, Msg: "自动登录失败，发送验证码失败，可能是请求过于频繁.", Step: 2}
				return errors.New("发送验证码失败，可能是请求过于频繁.")
			}
			l.ErrMsg = struct {
				Code int
				Msg  string
				Step int
			}{Code: 2004, Msg: "自动登录失败，账号已开启设备锁.需要验证码", Step: 2}
			return errors.Errorf("账号已开启设备锁,  向手机 %v 发送短信验证码.", res.SMSPhone)
		case client.UnsafeDeviceError:
			log.Warnf("账号已开启设备锁，请前往 -> %v <- 验证后重启Bot.", res.VerifyUrl)
			log.Infof("按 Enter 或等待 5s 后继续....")
			l.ErrMsg = struct {
				Code int
				Msg  string
				Step int
			}{Code: 2005, Msg: "自动登录失败，账号已开启设备锁.验证", Step: 2}
			return errors.Errorf("账号已开启设备锁，请前往 -> %v <- 验证后重启Bot.", res.VerifyUrl)
		case client.OtherLoginError, client.UnknownLoginError, client.TooManySMSRequestError:
			msg := res.ErrorMessage
			if strings.Contains(msg, "版本") {
				msg = "密码错误或账号被冻结"
				l.ErrMsg = struct {
					Code int
					Msg  string
					Step int
				}{Code: 2006, Msg: "自动登录失败，密码错误或账号被冻结", Step: 2}
				return errors.New("密码错误或账号被冻结")
			}
			if strings.Contains(msg, "冻结") {
				l.ErrMsg = struct {
					Code int
					Msg  string
					Step int
				}{Code: 2007, Msg: "自动登录失败，账号被冻结", Step: 2}
				return errors.New("密码错误或账号被冻结")
				//log.Fatalf("账号被冻结")
			}
			log.Warnf("登录失败: %v", msg)
			l.ErrMsg = struct {
				Code int
				Msg  string
				Step int
			}{Code: 2008, Msg: fmt.Sprintf("登录失败: %v", msg), Step: 2}
			return errors.Errorf("登录失败: %v", msg)
		}
	}
}
