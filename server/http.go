package server

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Mrs4s/MiraiGo/utils"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"gopkg.in/yaml.v3"

	"github.com/Mrs4s/go-cqhttp/coolq"
	"github.com/Mrs4s/go-cqhttp/global"
	"github.com/Mrs4s/go-cqhttp/modules/api"
	"github.com/Mrs4s/go-cqhttp/modules/config"
	"github.com/Mrs4s/go-cqhttp/modules/filter"
)

// HTTPServer HTTP通信相关配置
type HTTPServer struct {
	Disabled    bool   `yaml:"disabled"`
	Address     string `yaml:"address"`
	Host        string `yaml:"host"`
	Port        int    `yaml:"port"`
	Timeout     int32  `yaml:"timeout"`
	LongPolling struct {
		Enabled      bool `yaml:"enabled"`
		MaxQueueSize int  `yaml:"max-queue-size"`
	} `yaml:"long-polling"`
	Post []httpServerPost `yaml:"post"`

	MiddleWares `yaml:"middlewares"`
}

type httpServerPost struct {
	URL             string  `yaml:"url"`
	Secret          string  `yaml:"secret"`
	MaxRetries      *uint64 `yaml:"max-retries"`
	RetriesInterval *uint64 `yaml:"retries-interval"`
}

type httpServer struct {
	api         *api.Caller
	accessToken string
}

// HTTPClient 反向HTTP上报客户端
type HTTPClient struct {
	bot             *coolq.CQBot
	secret          string
	addr            string
	filter          string
	apiPort         int
	timeout         int32
	client          *http.Client
	MaxRetries      uint64
	RetriesInterval uint64
}

type httpCtx struct {
	json     gjson.Result
	query    url.Values
	postForm url.Values
}

const httpDefault = `
  - http: # HTTP 通信设置
      address: 0.0.0.0:5700 # HTTP监听地址
      timeout: 5      # 反向 HTTP 超时时间, 单位秒，<5 时将被忽略
      long-polling:   # 长轮询拓展
        enabled: false       # 是否开启
        max-queue-size: 2000 # 消息队列大小，0 表示不限制队列大小，谨慎使用
      middlewares:
        <<: *default # 引用默认中间件
      post:           # 反向HTTP POST地址列表
      #- url: ''                # 地址
      #  secret: ''             # 密钥
      #  max-retries: 3         # 最大重试，0 时禁用
      #  retries-interval: 1500 # 重试时间，单位毫秒，0 时立即
      #- url: http://127.0.0.1:5701/ # 地址
      #  secret: ''                  # 密钥
      #  max-retries: 10             # 最大重试，0 时禁用
      #  retries-interval: 1000      # 重试时间，单位毫秒，0 时立即
`

func init() {
	config.AddServer(&config.Server{Brief: "HTTP通信", Default: httpDefault})
}

var joinQuery = regexp.MustCompile(`\[(.+?),(.+?)]\.0`)

func (h *httpCtx) get(s string, join bool) gjson.Result {
	// support gjson advanced syntax:
	// h.Get("[a,b].0") see usage in http_test.go
	if join && joinQuery.MatchString(s) {
		matched := joinQuery.FindStringSubmatch(s)
		if r := h.get(matched[1], false); r.Exists() {
			return r
		}
		return h.get(matched[2], false)
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

func (h *httpCtx) Get(s string) gjson.Result {
	j := h.json.Get(s)
	if j.Exists() {
		return j
	}
	return h.get(s, true)
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
	if request.URL.Path == "/" {
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

func puint64Operator(p *uint64, def uint64) uint64 {
	if p == nil {
		return def
	}
	return *p
}

// runHTTP 启动HTTP服务器与HTTP上报客户端
func runHTTP(bot *coolq.CQBot, node yaml.Node) {
	var conf HTTPServer
	switch err := node.Decode(&conf); {
	case err != nil:
		log.Warn("读取http配置失败 :", err)
		fallthrough
	case conf.Disabled:
		return
	}

	network, addr := "tcp", conf.Address
	s := &httpServer{accessToken: conf.AccessToken}
	switch {
	case conf.Address != "":
		uri, err := url.Parse(conf.Address)
		if err == nil && uri.Scheme != "" {
			network = uri.Scheme
			addr = uri.Host + uri.Path
		}
	case conf.Host != "" || conf.Port != 0:
		addr = fmt.Sprintf("%s:%d", conf.Host, conf.Port)
		log.Warnln("HTTP 服务器使用了过时的配置格式，请更新配置文件！")
	default:
		goto client
	}
	s.api = api.NewCaller(bot)
	if conf.RateLimit.Enabled {
		s.api.Use(rateLimit(conf.RateLimit.Frequency, conf.RateLimit.Bucket))
	}
	if conf.LongPolling.Enabled {
		s.api.Use(longPolling(bot, conf.LongPolling.MaxQueueSize))
	}
	go func() {
		listener, err := net.Listen(network, addr)
		if err != nil {
			log.Infof("HTTP 服务启动失败, 请检查端口是否被占用: %v", err)
			log.Warnf("将在五秒后退出.")
			time.Sleep(time.Second * 5)
			os.Exit(1)
		}
		log.Infof("CQ HTTP 服务器已启动: %v", listener.Addr())
		log.Fatal(http.Serve(listener, s))
	}()
client:
	for _, c := range conf.Post {
		if c.URL != "" {
			go HTTPClient{
				bot:             bot,
				secret:          c.Secret,
				addr:            c.URL,
				apiPort:         conf.Port,
				filter:          conf.Filter,
				timeout:         conf.Timeout,
				MaxRetries:      puint64Operator(c.MaxRetries, 3),
				RetriesInterval: puint64Operator(c.RetriesInterval, 1500),
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
	rawAddress := c.addr
	network, address := resolveURI(c.addr)
	client := &http.Client{
		Timeout: time.Second * time.Duration(c.timeout),
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, addr string) (net.Conn, error) {
				if network == "unix" {
					host, _, err := net.SplitHostPort(addr)
					if err != nil {
						host = addr
					}
					filepath, err := base64.RawURLEncoding.DecodeString(host)
					if err == nil {
						addr = string(filepath)
					}
				}
				return net.Dial(network, addr)
			},
		},
	}
	c.addr = address // clean path
	c.client = client
	log.Infof("HTTP POST上报器已启动: %v", rawAddress)
	c.bot.OnEventPush(c.onBotPushEvent)
}

func (c *HTTPClient) onBotPushEvent(e *coolq.Event) {
	if c.filter != "" {
		flt := filter.Find(c.filter)
		if flt != nil && !flt.Eval(gjson.Parse(e.JSONString())) {
			log.Debugf("上报Event %v 到 HTTP 服务器 %s 时被过滤.", c.addr, e.JSONBytes())
			return
		}
	}

	header := make(http.Header)
	header.Set("X-Self-ID", strconv.FormatInt(c.bot.Client.Uin, 10))
	header.Set("User-Agent", "CQHttp/4.15.0")
	header.Set("Content-Type", "application/json")
	if c.secret != "" {
		mac := hmac.New(sha1.New, []byte(c.secret))
		_, _ = mac.Write(e.JSONBytes())
		header.Set("X-Signature", "sha1="+hex.EncodeToString(mac.Sum(nil)))
	}
	if c.apiPort != 0 {
		header.Set("X-API-Port", strconv.FormatInt(int64(c.apiPort), 10))
	}

	var req *http.Request
	var res *http.Response
	var err error
	for i := uint64(0); i <= c.MaxRetries; i++ {
		// see https://stackoverflow.com/questions/31337891/net-http-http-contentlength-222-with-body-length-0
		// we should create a new request for every single post trial
		req, err = http.NewRequest("POST", c.addr, bytes.NewReader(e.JSONBytes()))
		if err != nil {
			log.Warnf("上报 Event 数据到 %v 时创建请求失败: %v", c.addr, err)
			return
		}
		req.Header = header
		res, err = c.client.Do(req)
		if err != nil {
			if i < c.MaxRetries {
				log.Warnf("上报 Event 数据到 %v 失败: %v 将进行第 %d 次重试", c.addr, err, i+1)
			} else {
				log.Warnf("上报 Event 数据 %s 到 %v 失败: %v 停止上报：已达重试上限", e.JSONBytes(), c.addr, err)
				return
			}
			time.Sleep(time.Millisecond * time.Duration(c.RetriesInterval))
		}
	}
	defer res.Body.Close()

	log.Debugf("上报Event数据 %s 到 %v", e.JSONBytes(), c.addr)
	r, err := io.ReadAll(res.Body)
	if err != nil {
		return
	}
	if gjson.ValidBytes(r) {
		c.bot.CQHandleQuickOperation(gjson.Parse(e.JSONString()), gjson.ParseBytes(r))
	}
}
