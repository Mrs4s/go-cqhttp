package login

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
	"github.com/Mrs4s/MiraiGo/client"
	"github.com/Mrs4s/go-cqhttp/coolq"
	"github.com/Mrs4s/go-cqhttp/global"
	"github.com/Mrs4s/go-cqhttp/internal/base"
	"github.com/Mrs4s/go-cqhttp/iris-admin/models"
	"github.com/Mrs4s/go-cqhttp/iris-admin/utils/common"
	"github.com/Mrs4s/go-cqhttp/iris-admin/utils/jump"
	config2 "github.com/Mrs4s/go-cqhttp/modules/config"
	"github.com/Mrs4s/go-cqhttp/modules/servers"
	"github.com/kataras/iris/v12"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"os"
	"strings"
	"time"
)

// web上操作登录qq

type Dologin struct {
	Cli     *client.QQClient            //qqcli
	Qrcode  *client.QRCodeLoginResponse //当前的二维码
	Captcha *client.QRCodeLoginResponse //当前的验证码
	Config  *config2.Config
	// 错误信息 异步获取
	ErrMsg struct {
		Code int
		Msg  string
		Step int
	}
	Status bool //是否已经启动过net server
	Conn   chan string
	Bot    *coolq.CQBot
}

func NewDologin() *Dologin {
	cfg, _ := models.GetQqConfig()
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
		//log.Infof("按 Enter 或等待 5s 后继续....")
		//readLineTimeout(time.Second*5, "")
		//os.Exit(0)
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
			if l.Cli.Online {
				l.saveToken()
				l.Cli.AllowSlider = true
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
				//servers.Run(coolq.NewQQBot(l.Cli))
				log.Info("资源初始化完成, 开始处理信息.")
				log.Info("アトリは、高性能ですから!")
			}
		}
	}
}

func (l *Dologin) LoginSuccess(ctx iris.Context) {
	l.Conn <- "loginsuccess"
	ctx.Redirect("/admin/qq/info")
	//jump.JumpSuccessForIris(ctx, common.Msg{
	//	Msg: "登录成功",
	//	Url: "/admin/info/qq_config",
	//})
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
