package server

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Mrs4s/MiraiGo/utils"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"gopkg.in/yaml.v3"

	"github.com/Mrs4s/go-cqhttp/coolq"
	"github.com/Mrs4s/go-cqhttp/global"
	"github.com/Mrs4s/go-cqhttp/internal/base"
	"github.com/Mrs4s/go-cqhttp/modules/api"
	"github.com/Mrs4s/go-cqhttp/modules/config"
	"github.com/Mrs4s/go-cqhttp/modules/filter"
)

type httpServer struct {
	HTTP        *http.Server
	api         *api.Caller
	accessToken string
}

// HTTPClient 反向HTTP上报客户端
type HTTPClient struct {
	bot     *coolq.CQBot
	secret  string
	addr    string
	filter  string
	apiPort int
	timeout int32
}

type httpCtx struct {
	json     gjson.Result
	query    url.Values
	postForm url.Values
}

func (h *httpCtx) Get(s string) gjson.Result {
	j := h.json.Get(s)
	if j.Exists() {
		return j
	}
	validJSONParam := func(p string) bool {
		return (strings.HasPrefix(p, "{") || strings.HasPrefix(p, "[")) && gjson.Valid(p)
	}
	if h.postForm != nil {
		if form := h.postForm.Get(s); form != "" {
			if validJSONParam(form) {
				return gjson.Result{Type: gjson.JSON, Raw: form}
			}
			return gjson.Result{Type: gjson.String, Str: form}
		}
	}
	if h.query != nil {
		if query := h.query.Get(s); query != "" {
			if validJSONParam(query) {
				return gjson.Result{Type: gjson.JSON, Raw: query}
			}
			return gjson.Result{Type: gjson.String, Str: query}
		}
	}
	return gjson.Result{}
}

func (s *httpServer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	var ctx httpCtx
	contentType := request.Header.Get("Content-Type")
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
			err := request.ParseForm()
			if err != nil {
				log.Warnf("已拒绝客户端 %v 的请求: %v", request.RemoteAddr, err)
				writer.WriteHeader(http.StatusBadRequest)
			}
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
	if status := checkAuth(request, s.accessToken); status != http.StatusOK {
		writer.WriteHeader(status)
		return
	}

	var response global.MSG
	if base.AcceptOneBot12HTTPEndPoint && request.URL.Path == "/" {
		action := strings.TrimSuffix(ctx.Get("action").Str, "_async")
		log.Debugf("HTTPServer接收到API调用: %v", action)
		response = s.api.Call(action, ctx.Get("params"))
	} else {
		action := strings.TrimPrefix(request.URL.Path, "/")
		action = strings.TrimSuffix(action, "_async")
		log.Debugf("HTTPServer接收到API调用: %v", action)
		response = s.api.Call(action, &ctx)
	}

	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(writer).Encode(response)
}

func checkAuth(req *http.Request, token string) int {
	if token == "" { // quick path
		return http.StatusOK
	}

	auth := req.Header.Get("Authorization")
	if auth == "" {
		auth = req.URL.Query().Get("access_token")
	} else {
		authN := strings.SplitN(auth, " ", 2)
		if len(authN) == 2 {
			auth = authN[1]
		}
	}

	switch auth {
	case token:
		return http.StatusOK
	case "":
		return http.StatusUnauthorized
	default:
		return http.StatusForbidden
	}
}

// runHTTP 启动HTTP服务器与HTTP上报客户端
func runHTTP(bot *coolq.CQBot, node yaml.Node) {
	var conf config.HTTPServer
	switch err := node.Decode(&conf); {
	case err != nil:
		log.Warn("读取http配置失败 :", err)
		fallthrough
	case conf.Disabled:
		return
	}

	var addr string
	s := &httpServer{accessToken: conf.AccessToken}
	if conf.Host == "" || conf.Port == 0 {
		goto client
	}
	addr = fmt.Sprintf("%s:%d", conf.Host, conf.Port)
	s.api = api.NewCaller(bot)
	if conf.RateLimit.Enabled {
		s.api.Use(rateLimit(conf.RateLimit.Frequency, conf.RateLimit.Bucket))
	}
	if conf.LongPolling.Enabled {
		s.api.Use(longPolling(bot, conf.LongPolling.MaxQueueSize))
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
			go HTTPClient{
				bot:     bot,
				secret:  c.Secret,
				addr:    c.URL,
				apiPort: conf.Port,
				filter:  conf.Filter,
				timeout: conf.Timeout,
			}.Run()
		}
	}
}

// Run 运行反向HTTP服务
func (c HTTPClient) Run() {
	filter.Add(c.filter)
	if c.timeout < 5 {
		c.timeout = 5
	}
	c.bot.OnEventPush(c.onBotPushEvent)
	log.Infof("HTTP POST上报器已启动: %v", c.addr)
}

func (c *HTTPClient) onBotPushEvent(e *coolq.Event) {
	if c.filter != "" {
		flt := filter.Find(c.filter)
		if flt != nil && !flt.Eval(gjson.Parse(e.JSONString())) {
			log.Debugf("上报Event %v 到 HTTP 服务器 %s 时被过滤.", c.addr, e.JSONBytes())
			return
		}
	}

	client := http.Client{Timeout: time.Second * time.Duration(c.timeout)}
	req, _ := http.NewRequest("POST", c.addr, bytes.NewReader(e.JSONBytes()))
	req.Header.Set("X-Self-ID", strconv.FormatInt(c.bot.Client.Uin, 10))
	req.Header.Set("User-Agent", "CQHttp/4.15.0")
	if c.secret != "" {
		mac := hmac.New(sha1.New, []byte(c.secret))
		_, _ = mac.Write(e.JSONBytes())
		req.Header.Set("X-Signature", "sha1="+hex.EncodeToString(mac.Sum(nil)))
	}
	if c.apiPort != 0 {
		req.Header.Set("X-API-Port", strconv.FormatInt(int64(c.apiPort), 10))
	}

	var res *http.Response
	var err error
	const maxAttemptTimes = 5

	for i := 0; i <= maxAttemptTimes; i++ {
		res, err = client.Do(req)
		if err == nil {
			//goland:noinspection GoDeferInLoop
			defer res.Body.Close()
			break
		}
		if i != maxAttemptTimes {
			log.Warnf("上报 Event 数据到 %v 失败: %v 将进行第 %d 次重试", c.addr, err, i+1)
		}
		const maxWait = int64(time.Second * 3)
		const minWait = int64(time.Millisecond * 500)
		wait := rand.Int63n(maxWait-minWait) + minWait
		time.Sleep(time.Duration(wait))
	}

	if err != nil {
		log.Warnf("上报Event数据 %s 到 %v 失败: %v", e.JSONBytes(), c.addr, err)
		return
	}
	log.Debugf("上报Event数据 %s 到 %v", e.JSONBytes(), c.addr)

	r, err := io.ReadAll(res.Body)
	if err != nil {
		return
	}
	if gjson.ValidBytes(r) {
		c.bot.CQHandleQuickOperation(gjson.Parse(e.JSONString()), gjson.ParseBytes(r))
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
