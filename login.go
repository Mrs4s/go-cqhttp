package main

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"os"
	"strings"
	"time"

	qrcodeTerminal "github.com/Baozisoftware/qrcode-terminal-go"
	"github.com/Mrs4s/MiraiGo/client"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/tuotoo/qrcode"

	"github.com/Mrs4s/go-cqhttp/global"
)

var console = bufio.NewReader(os.Stdin)

var readLine = func() (str string) {
	str, _ = console.ReadString('\n')
	str = strings.TrimSpace(str)
	return
}

var readLineTimeout = func(t time.Duration, de string) (str string) {
	r := make(chan string)
	go func() {
		select {
		case r <- readLine():
		case <-time.After(t):
		}
	}()
	str = de
	select {
	case str = <-r:
	case <-time.After(t):
	}
	return
}

var cli *client.QQClient

// ErrSMSRequestError SMS请求出错
var ErrSMSRequestError = errors.New("sms request error")

func commonLogin() error {
	res, err := cli.Login()
	if err != nil {
		return err
	}
	return loginResponseProcessor(res)
}

func qrcodeLogin() error {
	rsp, err := cli.FetchQRCode()
	if err != nil {
		return err
	}
	fi, err := qrcode.Decode(bytes.NewReader(rsp.ImageData))
	if err != nil {
		return err
	}
	_ = ioutil.WriteFile("qrcode.png", rsp.ImageData, 0644)
	defer func() { _ = os.Remove("qrcode.png") }()
	log.Infof("请使用手机QQ扫描二维码 (qrcode.png) : ")
	time.Sleep(time.Second)
	qrcodeTerminal.New().Get(fi.Content).Print()
	s, err := cli.QueryQRCodeStatus(rsp.Sig)
	if err != nil {
		return err
	}
	prevState := s.State
	for {
		time.Sleep(time.Second)
		s, _ = cli.QueryQRCodeStatus(rsp.Sig)
		if s == nil {
			continue
		}
		if prevState == s.State {
			continue
		}
		prevState = s.State
		if s.State == client.QRCodeCanceled {
			log.Fatalf("扫码被用户取消.")
		}
		if s.State == client.QRCodeTimeout {
			log.Fatalf("二维码过期")
		}
		if s.State == client.QRCodeWaitingForConfirm {
			log.Infof("扫码成功, 请在手机端确认登录.")
		}
		if s.State == client.QRCodeConfirmed {
			res, err := cli.QRCodeLogin(s.LoginInfo)
			if err != nil {
				return err
			}
			return loginResponseProcessor(res)
		}
	}
}

func loginResponseProcessor(res *client.LoginResponse) error {
	var err error
	for {
		if err != nil {
			return err
		}
		if res.Success {
			return nil
		}
		var text string
		switch res.Error {
		case client.SliderNeededError:
			log.Warnf("登录需要滑条验证码. ")
			log.Warnf("请参考文档 -> https://github.com/Mrs4s/go-cqhttp/blob/master/docs/slider.md <- 进行处理")
			log.Warnf("1. 自行抓包并获取 Ticket 输入.")
			log.Warnf("2. 使用手机QQ扫描二维码登入. (推荐)")
			log.Warn("请输入(1 - 2) (将在10秒后自动选择2)：")
			text = readLineTimeout(time.Second*10, "2")
			if strings.Contains(text, "1") {
				println()
				log.Warnf("请用浏览器打开 -> %v <- 并获取Ticket.", res.VerifyUrl)
				println()
				log.Warn("请输入Ticket： (Enter 提交)")
				text = readLine()
				res, err = cli.SubmitTicket(text)
				continue
			}
			return qrcodeLogin()
		case client.NeedCaptcha:
			log.Warnf("登录需要验证码.")
			_ = ioutil.WriteFile("captcha.jpg", res.CaptchaImage, 0644)
			log.Warnf("请输入验证码 (captcha.jpg)： (Enter 提交)")
			text = readLine()
			global.DelFile("captcha.jpg")
			res, err = cli.SubmitCaptcha(text, res.CaptchaSign)
			continue
		case client.SMSNeededError:
			log.Warnf("账号已开启设备锁, 按 Enter 向手机 %v 发送短信验证码.", res.SMSPhone)
			readLine()
			if !cli.RequestSMS() {
				log.Warnf("发送验证码失败，可能是请求过于频繁.")
				return errors.WithStack(ErrSMSRequestError)
			}
			log.Warn("请输入短信验证码： (Enter 提交)")
			text = readLine()
			res, err = cli.SubmitSMS(text)
			continue
		case client.SMSOrVerifyNeededError:
			log.Warnf("账号已开启设备锁，请选择验证方式:")
			log.Warnf("1. 向手机 %v 发送短信验证码", res.SMSPhone)
			log.Warnf("2. 使用手机QQ扫码验证.")
			log.Warn("请输入(1 - 2) (将在10秒后自动选择2)：")
			text = readLineTimeout(time.Second*10, "2")
			if strings.Contains(text, "1") {
				if !cli.RequestSMS() {
					log.Warnf("发送验证码失败，可能是请求过于频繁.")
					return errors.WithStack(ErrSMSRequestError)
				}
				log.Warn("请输入短信验证码： (Enter 提交)")
				text = readLine()
				res, err = cli.SubmitSMS(text)
				continue
			}
			fallthrough
		case client.UnsafeDeviceError:
			log.Warnf("账号已开启设备锁，请前往 -> %v <- 验证后重启Bot.", res.VerifyUrl)
			log.Infof("按 Enter 或等待 5s 后继续....")
			readLineTimeout(time.Second*5, "")
			os.Exit(0)
		case client.OtherLoginError, client.UnknownLoginError, client.TooManySMSRequestError:
			msg := res.ErrorMessage
			if strings.Contains(msg, "版本") {
				msg = "密码错误或账号被冻结"
			}
			if strings.Contains(msg, "冻结") {
				log.Fatalf("账号被冻结")
			}
			log.Warnf("登录失败: %v", msg)
			log.Infof("按 Enter 或等待 5s 后继续....")
			readLineTimeout(time.Second*5, "")
			os.Exit(0)
		}
	}
}
