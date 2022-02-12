package gocq

import (
	"bufio"
	"bytes"
	"image"
	"image/png"
	"os"
	"strings"
	"time"

	"github.com/Mrs4s/MiraiGo/client"
	"github.com/mattn/go-colorable"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/Mrs4s/go-cqhttp/global"
)

var console = bufio.NewReader(os.Stdin)

func readLine() (str string) {
	str, _ = console.ReadString('\n')
	str = strings.TrimSpace(str)
	return
}

func readLineTimeout(t time.Duration, de string) (str string) {
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

func printQRCode(imgData []byte) {
	const (
		black = "\033[48;5;0m  \033[0m"
		white = "\033[48;5;7m  \033[0m"
	)
	img, err := png.Decode(bytes.NewReader(imgData))
	if err != nil {
		log.Panic(err)
	}
	data := img.(*image.Gray).Pix
	bound := img.Bounds().Max.X
	buf := make([]byte, 0, (bound*4+1)*(bound))
	i := 0
	for y := 0; y < bound; y++ {
		i = y * bound
		for x := 0; x < bound; x++ {
			if data[i] != 255 {
				buf = append(buf, white...)
			} else {
				buf = append(buf, black...)
			}
			i++
		}
		buf = append(buf, '\n')
	}
	_, _ = colorable.NewColorableStdout().Write(buf)
}

func qrcodeLogin() error {
	rsp, err := cli.FetchQRCodeCustomSize(1, 2, 1)
	if err != nil {
		return err
	}
	_ = os.WriteFile("qrcode.png", rsp.ImageData, 0o644)
	defer func() { _ = os.Remove("qrcode.png") }()
	if cli.Uin != 0 {
		log.Infof("请使用账号 %v 登录手机QQ扫描二维码 (qrcode.png) : ", cli.Uin)
	} else {
		log.Infof("请使用手机QQ扫描二维码 (qrcode.png) : ")
	}
	time.Sleep(time.Second)
	printQRCode(rsp.ImageData)
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
		switch s.State {
		case client.QRCodeCanceled:
			log.Fatalf("扫码被用户取消.")
		case client.QRCodeTimeout:
			log.Fatalf("二维码过期")
		case client.QRCodeWaitingForConfirm:
			log.Infof("扫码成功, 请在手机端确认登录.")
		case client.QRCodeConfirmed:
			res, err := cli.QRCodeLogin(s.LoginInfo)
			if err != nil {
				return err
			}
			return loginResponseProcessor(res)
		case client.QRCodeImageFetch, client.QRCodeWaitingForScan:
			// ignore
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
			log.Warnf("登录需要滑条验证码, 请使用手机QQ扫描二维码以继续登录.")
			cli.Disconnect()
			cli.Release()
			cli = client.NewClientEmpty()
			return qrcodeLogin()
		case client.NeedCaptcha:
			log.Warnf("登录需要验证码.")
			_ = os.WriteFile("captcha.jpg", res.CaptchaImage, 0o644)
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
