package server

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Mrs4s/MiraiGo/utils"
	"github.com/guonaihong/gout"
	"github.com/guonaihong/gout/dataflow"
	jsoniter "github.com/json-iterator/go"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"

	"github.com/Mrs4s/go-cqhttp/coolq"
	"github.com/Mrs4s/go-cqhttp/global/config"
)

type httpServer struct {
	HTTP        *http.Server
	api         *apiCaller
	accessToken string
}

var json = jsoniter.ConfigCompatibleWithStandardLibrary

// HTTPClient 反向HTTP上报客户端
type HTTPClient struct {
	bot     *coolq.CQBot
	secret  string
	addr    string
	filter  string
	timeout int32
}

type httpCtx struct {
	json     gjson.Result
	query    url.Values
	postForm url.Values

	headerAuth string
}

func (h *httpCtx) Get(s string) gjson.Result {
	j := h.json.Get(s)
	if j.Exists() {
		return j
	}
	if h.postForm != nil {
		if form := h.postForm.Get(s); form != "" {
			return gjson.Result{Type: gjson.String, Str: form}
		}
	}
	if h.query != nil {
		if query := h.query.Get(s); query != "" {
			return gjson.Result{Type: gjson.String, Str: query}
		}
	}
	return gjson.Result{}
}

func (s *httpServer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	var ctx httpCtx
	contentType := request.Header.Get("Content-Type")
	ctx.headerAuth = request.Header.Get("Authorization")
	switch request.Method {
	case http.MethodPost:
		if strings.Contains(contentType, "application/json") {
			body, err := io.ReadAll(request.Body)
			if err != nil {
				log.Warnf("获取请求 %v 的Body时出现错误: %v", request.RequestURI, err)
				writer.WriteHeader(http.StatusBadRequest)
				return
			}
			if !gjson.ValidBytes(body) {
				log.Warnf("已拒绝客户端 %v 的请求: 非法Json", request.RemoteAddr)
				writer.WriteHeader(http.StatusBadRequest)
				return
			}
			ctx.json = gjson.Parse(utils.B2S(body))
		}
		if strings.Contains(contentType, "application/x-www-form-urlencoded") {
			ctx.postForm = request.PostForm
		}
		fallthrough
	case http.MethodGet:
		ctx.query = request.URL.Query()

	default:
		log.Warnf("已拒绝客户端 %v 的请求: 方法错误", request.RemoteAddr)
		writer.WriteHeader(http.StatusNotFound)
		return
	}

	if s.accessToken != "" {
		if status := checkAuth(ctx, s.accessToken); status != http.StatusOK {
			writer.WriteHeader(status)
			return
		}
	}

	action := strings.TrimPrefix(request.URL.Path, "/")
	action = strings.TrimSuffix(action, "_async")
	log.Debugf("HTTPServer接收到API调用: %v", action)
	ret := s.api.callAPI(action, &ctx)

	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(writer).Encode(ret)
}

func checkAuth(ctx httpCtx, token string) int {
	auth := ctx.headerAuth
	if auth == "" {
		auth = ctx.query.Get("access_token")
	} else {
		authN := strings.SplitN(auth, " ", 2)
		if len(authN) == 2 {
			auth = authN[1]
		}
	}

	switch {
	case auth == "":
		return http.StatusUnauthorized
	case auth != token:
		return http.StatusForbidden
	default:
		return http.StatusOK
	}
}

// RunHTTPServerAndClients 启动HTTP服务器与HTTP上报客户端
func RunHTTPServerAndClients(bot *coolq.CQBot, conf *config.HTTPServer) {
	if conf.Disabled {
		return
	}
	var (
		s    = new(httpServer)
		addr string
	)
	s.accessToken = conf.AccessToken
	if conf.Host == "" || conf.Port == 0 {
		goto client
	}
	addr = fmt.Sprintf("%s:%d", conf.Host, conf.Port)
	s.api = newAPICaller(bot)
	if conf.RateLimit.Enabled {
		s.api.use(rateLimit(conf.RateLimit.Frequency, conf.RateLimit.Bucket))
	}

	go func() {
		log.Infof("CQ HTTP 服务器已启动: %v", addr)
		s.HTTP = &http.Server{
			Addr:    addr,
			Handler: s,
		}
		if err := s.HTTP.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error(err)
			log.Infof("HTTP 服务启动失败, 请检查端口是否被占用.")
			log.Warnf("将在五秒后退出.")
			time.Sleep(time.Second * 5)
			os.Exit(1)
		}
	}()
client:
	for _, c := range conf.Post {
		if c.URL != "" {
			go newHTTPClient().Run(c.URL, c.Secret, conf.Filter, conf.Timeout, bot)
		}
	}
}

// newHTTPClient 返回反向HTTP客户端
func newHTTPClient() *HTTPClient {
	return &HTTPClient{}
}

// Run 运行反向HTTP服务
func (c *HTTPClient) Run(addr, secret, filter string, timeout int32, bot *coolq.CQBot) {
	c.bot = bot
	c.secret = secret
	c.addr = addr
	c.timeout = timeout
	c.filter = filter
	addFilter(filter)
	if c.timeout < 5 {
		c.timeout = 5
	}
	bot.OnEventPush(c.onBotPushEvent)
	log.Infof("HTTP POST上报器已启动: %v", addr)
}

func (c *HTTPClient) onBotPushEvent(m *bytes.Buffer) {
	var res string
	if c.filter != "" {
		filter := findFilter(c.filter)
		if filter != nil && !filter.Eval(gjson.Parse(utils.B2S(m.Bytes()))) {
			log.Debugf("上报Event %v 到 HTTP 服务器 %v 时被过滤.", c.addr, utils.B2S(m.Bytes()))
			return
		}
	}

	err := gout.POST(c.addr).SetJSON(m.Bytes()).BindBody(&res).SetHeader(func() gout.H {
		h := gout.H{
			"X-Self-ID":  c.bot.Client.Uin,
			"User-Agent": "CQHttp/4.15.0",
		}
		if c.secret != "" {
			mac := hmac.New(sha1.New, []byte(c.secret))
			_, err := mac.Write(m.Bytes())
			if err != nil {
				log.Error(err)
				return nil
			}
			h["X-Signature"] = "sha1=" + hex.EncodeToString(mac.Sum(nil))
		}
		return h
	}()).SetTimeout(time.Second * time.Duration(c.timeout)).F().Retry().Attempt(5).
		WaitTime(time.Millisecond * 500).MaxWaitTime(time.Second * 5).
		Func(func(con *dataflow.Context) error {
			if con.Error != nil {
				log.Warnf("上报Event到 HTTP 服务器 %v 时出现错误: %v 将重试.", c.addr, con.Error)
				return con.Error
			}
			return nil
		}).Do()
	if err != nil {
		log.Warnf("上报Event数据 %v 到 %v 失败: %v", utils.B2S(m.Bytes()), c.addr, err)
		return
	}
	log.Debugf("上报Event数据 %v 到 %v", utils.B2S(m.Bytes()), c.addr)
	if gjson.Valid(res) {
		c.bot.CQHandleQuickOperation(gjson.Parse(utils.B2S(m.Bytes())), gjson.Parse(res))
	}
}

func (s *httpServer) ShutDown() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.HTTP.Shutdown(ctx); err != nil {
		log.Fatal("http Server Shutdown:", err)
	}
	<-ctx.Done()
	log.Println("timeout of 5 seconds.")
	log.Println("http Server exiting")
}
