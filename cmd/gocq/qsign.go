package gocq

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"

	"github.com/Mrs4s/MiraiGo/utils"
	"github.com/Mrs4s/go-cqhttp/global"
	"github.com/Mrs4s/go-cqhttp/internal/base"
	"github.com/Mrs4s/go-cqhttp/internal/download"
	"github.com/Mrs4s/go-cqhttp/modules/config"
)

// 配置的所有签名服务器
var signServers []config.SignServer
var checkLock sync.Mutex

// 当前签名服务器
var currentSignServer config.SignServer
var currentOK atomic.Bool

var errorCount = int64(0)

// GetAvaliableSignServer 获取可用的签名服务器，没有则返回空和相应错误
func GetAvaliableSignServer() (config.SignServer, error) {
	if currentOK.Load() {
		return currentSignServer, nil
	}
	maxCount := base.Account.MaxCheckCount
	signServers = base.SignServers
	if maxCount == 0 && atomic.LoadInt64(&errorCount) > 3 {
		currentSignServer = signServers[0]
		currentOK.Store(true)
		return currentSignServer, nil
	}
	if maxCount > 0 && atomic.LoadInt64(&errorCount) > int64(maxCount) {
		log.Fatalf("获取可用签名服务器失败次数超过 %v 次, 正在离线", maxCount)
	}
	if len(currentSignServer.URL) > 1 {
		log.Warnf("当前签名服务器 %v 不可用，正在查找可用服务器", currentSignServer.URL)
	}
	if checkLock.TryLock() {
		defer checkLock.Unlock()
		result := make(chan config.SignServer)
		for _, server := range signServers {
			go func(s config.SignServer, r chan config.SignServer) {
				if s.URL == "-" || s.URL == "" {
					return
				}
				if isServerAvaliable(s.URL) {
					errorCount = 0
					result <- s
					log.Infof("使用签名服务器 url=%v, key=%v, auth=%v", s.URL, s.Key, s.Authorization)
				}
			}(server, result)
		}
		select {
		case res := <-result:
			currentSignServer = res
			currentOK.Store(true)
			signRegister(base.Account.Uin, device.AndroidId, device.Guid, device.QImei36, res.Key) // 注册实例
			return currentSignServer, nil
		case <-time.After(time.Duration(base.SignServerTimeout) * time.Second):
			return config.SignServer{}, errors.New(
				fmt.Sprintf("no avaliable sign-server, timeout=%v", base.SignServerTimeout))
		}
	}
	return config.SignServer{}, errors.New("checking sign-servers")
}

func isServerAvaliable(signServer string) bool {
	resp, err := download.Request{
		Method: http.MethodGet,
		URL:    signServer,
	}.WithTimeout(5 * time.Second).Bytes()
	if err == nil && gjson.GetBytes(resp, "code").Int() == 0 {
		return true
	}
	return false
}

/*
请求签名服务器

	url: api + params 组合的字符串，无须包含签名服务器地址
	return: signServer, response, error
*/
func requestSignServer(method string, url string, headers map[string]string, body io.Reader) (string, []byte, error) {
	signServer, e := GetAvaliableSignServer()
	if e != nil && len(signServer.URL) <= 1 { // 没有可用的
		log.Warnf("获取可用签名服务器出错：%v", e)
		atomic.AddInt64(&errorCount, 1)
		signServer = signServers[0] // 没有获取到时使用第一个
	}
	if !strings.HasPrefix(url, signServer.URL) {
		url = strings.TrimSuffix(signServer.URL, "/") + "/" + strings.TrimPrefix(url, "/")
	}
	if headers == nil {
		headers = map[string]string{}
	}
	auth := signServer.Authorization
	if auth != "-" && auth != "" {
		headers["Authorization"] = auth
	}
	req := download.Request{
		Method: method,
		Header: headers,
		URL:    url,
		Body:   body,
	}.WithTimeout(time.Duration(base.SignServerTimeout) * time.Second)
	resp, err := req.Bytes()
	if err != nil {
		currentOK.Store(false)
	}
	return signServer.URL, resp, err
}

func energy(uin uint64, id string, _ string, salt []byte) ([]byte, error) {
	url := "custom_energy" + fmt.Sprintf("?data=%v&salt=%v&uin=%v&android_id=%v&guid=%v",
		id, hex.EncodeToString(salt), uin, utils.B2S(device.AndroidId), hex.EncodeToString(device.Guid))
	if base.IsBelow110 {
		url = "custom_energy" + fmt.Sprintf("?data=%v&salt=%v", id, hex.EncodeToString(salt))
	}
	signServer, response, err := requestSignServer(http.MethodGet, url, nil, nil)
	if err != nil {
		log.Warnf("获取T544 sign时出现错误: %v. server: %v", err, signServer)
		return nil, err
	}
	data, err := hex.DecodeString(gjson.GetBytes(response, "data").String())
	if err != nil {
		log.Warnf("获取T544 sign时出现错误: %v", err)
		return nil, err
	}
	if len(data) == 0 {
		log.Warnf("获取T544 sign时出现错误: %v.", "data is empty")
		return nil, errors.New("data is empty")
	}
	return data, nil
}

// signSubmit
// 提交回调 buffer
func signSubmit(uin string, cmd string, callbackID int64, buffer []byte, t string) {
	buffStr := hex.EncodeToString(buffer)
	tail := 64
	endl := "..."
	if len(buffStr) < tail {
		tail = len(buffStr)
		endl = "."
	}
	log.Infof("submit %v: uin=%v, cmd=%v, callbackID=%v, buffer=%v%s", t, uin, cmd, callbackID, buffStr[:tail], endl)

	signServer, _, err := requestSignServer(
		http.MethodGet,
		"submit"+fmt.Sprintf("?uin=%v&cmd=%v&callback_id=%v&buffer=%v",
			uin, cmd, callbackID, buffStr),
		nil, nil,
	)
	if err != nil {
		log.Warnf("提交 callback 时出现错误: %v. server: %v", err, signServer)
	}
}

// signCallback
// 刷新 token 和签名的回调
func signCallback(uin string, results []gjson.Result, t string) {
	for _, result := range results {
		cmd := result.Get("cmd").String()
		callbackID := result.Get("callbackId").Int()
		body, _ := hex.DecodeString(result.Get("body").String())
		ret, err := cli.SendSsoPacket(cmd, body)
		if err != nil || len(ret) == 0 {
			log.Warnf("Callback error: %v, Or response data is empty", err)
			continue // 发送 SsoPacket 出错或返回数据为空时跳过
		}
		signSubmit(uin, cmd, callbackID, ret, t)
	}
}

func signRequset(seq uint64, uin string, cmd string, qua string, buff []byte) (sign []byte, extra []byte, token []byte, err error) {
	headers := map[string]string{"Content-Type": "application/x-www-form-urlencoded"}
	_, response, err := requestSignServer(
		http.MethodPost,
		"sign",
		headers,
		bytes.NewReader([]byte(fmt.Sprintf("uin=%v&qua=%s&cmd=%s&seq=%v&buffer=%v&android_id=%v&guid=%v",
			uin, qua, cmd, seq, hex.EncodeToString(buff), utils.B2S(device.AndroidId), hex.EncodeToString(device.Guid)))),
	)
	if err != nil {
		return nil, nil, nil, err
	}
	sign, _ = hex.DecodeString(gjson.GetBytes(response, "data.sign").String())
	extra, _ = hex.DecodeString(gjson.GetBytes(response, "data.extra").String())
	token, _ = hex.DecodeString(gjson.GetBytes(response, "data.token").String())
	if !base.IsBelow110 {
		go signCallback(uin, gjson.GetBytes(response, "data.requestCallback").Array(), "sign")
	}
	return sign, extra, token, nil
}

var registerLock sync.Mutex

func signRegister(uin int64, androidID, guid []byte, qimei36, key string) {
	if base.IsBelow110 {
		log.Warn("签名服务器版本低于1.1.0, 跳过实例注册")
		return
	}
	signServer, resp, err := requestSignServer(
		http.MethodGet,
		"register"+fmt.Sprintf("?uin=%v&android_id=%v&guid=%v&qimei36=%v&key=%s",
			uin, utils.B2S(androidID), hex.EncodeToString(guid), qimei36, key),
		nil, nil,
	)
	if err != nil {
		log.Warnf("注册QQ实例时出现错误: %v. server: %v", err, signServer)
		return
	}
	msg := gjson.GetBytes(resp, "msg")
	if gjson.GetBytes(resp, "code").Int() != 0 {
		log.Warnf("注册QQ实例时出现错误: %v. server: %v", msg, signServer)
		return
	}
	log.Infof("注册QQ实例 %v 成功: %v", uin, msg)
}

func signRefreshToken(uin string) error {
	log.Info("正在刷新 token")
	_, resp, err := requestSignServer(
		http.MethodGet,
		"request_token?uin="+uin,
		nil, nil,
	)
	if err != nil {
		return err
	}
	msg := gjson.GetBytes(resp, "msg")
	code := gjson.GetBytes(resp, "code")
	if code.Int() != 0 {
		return errors.New("code=" + code.String() + ", msg: " + msg.String())
	}
	go signCallback(uin, gjson.GetBytes(resp, "data").Array(), "request token")
	return nil
}

var missTokenCount = uint64(0)
var lastToken = ""

func sign(seq uint64, uin string, cmd string, qua string, buff []byte) (sign []byte, extra []byte, token []byte, err error) {
	i := 0
	for {
		sign, extra, token, err = signRequset(seq, uin, cmd, qua, buff)
		if err != nil {
			log.Warnf("获取sso sign时出现错误: %v. server: %v", err, currentSignServer.URL)
		}
		if i > 0 {
			break
		}
		i++
		if (!base.IsBelow110) && base.Account.AutoRegister && err == nil && len(sign) == 0 {
			if registerLock.TryLock() { // 避免并发时多处同时销毁并重新注册
				log.Warn("获取签名为空，实例可能丢失，正在尝试重新注册")
				defer registerLock.Unlock()
				err := signServerDestroy(uin)
				if err != nil {
					log.Warnln(err) // 实例真的丢失时则必出错，或许应该不 return , 以重新获取本次签名
					// return nil, nil, nil, err
				}
				signRegister(base.Account.Uin, device.AndroidId, device.Guid, device.QImei36, currentSignServer.Key)
			}
			continue
		}
		if (!base.IsBelow110) && base.Account.AutoRefreshToken && len(token) == 0 {
			log.Warnf("token 已过期, 总丢失 token 次数为 %v", atomic.AddUint64(&missTokenCount, 1))
			if registerLock.TryLock() {
				defer registerLock.Unlock()
				if err := signRefreshToken(uin); err != nil {
					log.Warnf("刷新 token 出现错误: %v. server: %v", err, currentSignServer.URL)
				} else {
					log.Info("刷新 token 成功")
				}
			}
			continue
		}
		break
	}
	if tokenString := hex.EncodeToString(token); lastToken != tokenString {
		log.Infof("token 已更新：%v -> %v", lastToken, tokenString)
		lastToken = tokenString
	}
	if len(sign) == 0 {
		currentOK.Store(false)
	}
	return sign, extra, token, err
}

func signServerDestroy(uin string) error {
	signServer, signVersion, err := signVersion()
	if err != nil {
		return errors.Wrapf(err, "获取签名服务版本出现错误, server: %v", signServer)
	}
	if global.VersionNameCompare("v"+signVersion, "v1.1.6") {
		return errors.Errorf("当前签名服务器版本 %v 低于 1.1.6，无法使用 destroy 接口", signVersion)
	}
	signServer, resp, err := requestSignServer(
		http.MethodGet,
		"destroy"+fmt.Sprintf("?uin=%v&key=%v", uin, currentSignServer.Key),
		nil, nil,
	)
	if err != nil || gjson.GetBytes(resp, "code").Int() != 0 {
		return errors.Wrapf(err, "destroy 实例出现错误, server: %v", signServer)
	}
	return nil
}

func signVersion() (signServer string, version string, err error) {
	signServer, resp, err := requestSignServer(http.MethodGet, "", nil, nil)
	if err != nil {
		return signServer, "", err
	}
	if gjson.GetBytes(resp, "code").Int() == 0 {
		return signServer, gjson.GetBytes(resp, "data.version").String(), nil
	}
	return signServer, "", errors.New("empty version")
}

// 定时刷新 token, interval 为间隔时间（分钟）
func signStartRefreshToken(interval int64) {
	if interval <= 0 {
		log.Warn("定时刷新 token 已关闭")
		return
	}
	log.Infof("每 %v 分钟将刷新一次签名 token", interval)
	if interval < 10 {
		log.Warnf("间隔时间 %v 分钟较短，推荐 30~40 分钟", interval)
	}
	if interval > 60 {
		log.Warn("间隔时间不能超过 60 分钟，已自动设置为 60 分钟")
		interval = 60
	}
	t := time.NewTicker(time.Duration(interval) * time.Minute)
	qqstr := strconv.FormatInt(base.Account.Uin, 10)
	defer t.Stop()
	for range t.C {
		if currentSignServer != signServers[0] && isServerAvaliable(signServers[0].URL) {
			currentSignServer = signServers[0]
			currentOK.Store(true)
			log.Infof("主签名服务器可用，已切换至主签名服务器 %v", currentSignServer.URL)
		}
		err := signRefreshToken(qqstr)
		if err != nil {
			log.Warnf("刷新 token 出现错误: %v. server: %v", err, currentSignServer.URL)
		}
	}
}
