package gocq

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Mrs4s/MiraiGo/utils"
	"github.com/Mrs4s/go-cqhttp/global"
	"github.com/Mrs4s/go-cqhttp/internal/base"
	"github.com/Mrs4s/go-cqhttp/internal/download"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

/*
请求签名服务器

	url: api + params 组合的字符串，无须包含签名服务器地址
	return: signServer, response, error
*/
func requestSignServer(method string, url string, headers map[string]string, body io.Reader) (string, []byte, error) {
	signServer := base.SignServer
	if !strings.HasPrefix(url, signServer) {
		url = strings.TrimSuffix(signServer, "/") + "/" + strings.TrimPrefix(url, "/")
	}
	if headers == nil {
		headers = map[string]string{}
	}
	signServerBearer := base.SignServerBearer
	if signServerBearer != "-" && signServerBearer != "" {
		headers["Authorization"] = "Bearer " + signServerBearer
	}
	req := download.Request{
		Method: method,
		Header: headers,
		URL:    url,
		Body:   body,
	}.WithTimeout(time.Duration(base.SignServerTimeout) * time.Second)
	resp, err := req.Bytes()
	return signServer, resp, err
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
			log.Warnf("获取sso sign时出现错误: %v. server: %v", err, base.SignServer)
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
				signRegister(base.Account.Uin, device.AndroidId, device.Guid, device.QImei36, base.Key)
			}
			continue
		}
		if (!base.IsBelow110) && base.Account.AutoRefreshToken && len(token) == 0 {
			log.Warnf("token 已过期, 总丢失 token 次数为 %v", atomic.AddUint64(&missTokenCount, 1))
			if registerLock.TryLock() {
				defer registerLock.Unlock()
				if err := signRefreshToken(uin); err != nil {
					log.Warnf("刷新 token 出现错误: %v. server: %v", err, base.SignServer)
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
		"destroy"+fmt.Sprintf("?uin=%v&key=%v", uin, base.Key),
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
		err := signRefreshToken(qqstr)
		if err != nil {
			log.Warnf("刷新 token 出现错误: %v. server: %v", err, base.SignServer)
		}
	}
}

func signWaitServer() bool {
	t := time.NewTicker(time.Second * 5)
	defer t.Stop()
	i := 0
	for range t.C {
		if i > 3 {
			return false
		}
		i++
		u, err := url.Parse(base.SignServer)
		if err != nil {
			log.Warnf("连接到签名服务器出现错误: %v", err)
			continue
		}
		r := utils.RunTCPPingLoop(u.Host, 4)
		if r.PacketsLoss > 0 {
			log.Warnf("连接到签名服务器出现错误: 丢包%d/%d 时延%dms", r.PacketsLoss, r.PacketsSent, r.AvgTimeMill)
			continue
		}
		break
	}
	log.Infof("连接至签名服务器: %s", base.SignServer)
	return true
}
