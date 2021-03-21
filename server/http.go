package server

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Mrs4s/go-cqhttp/coolq"
	"github.com/Mrs4s/go-cqhttp/global"

	"github.com/Mrs4s/MiraiGo/utils"
	"github.com/gin-gonic/gin"
	"github.com/guonaihong/gout"
	"github.com/guonaihong/gout/dataflow"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

type httpServer struct {
	engine *gin.Engine
	bot    *coolq.CQBot
	HTTP   *http.Server
	api    apiCaller
}

// HTTPClient 反向HTTP上报客户端
type HTTPClient struct {
	bot     *coolq.CQBot
	secret  string
	addr    string
	timeout int32
}

type httpContext struct {
	ctx *gin.Context
}

// CQHTTPApiServer CQHTTPApiServer实例
var CQHTTPApiServer = &httpServer{}

// Debug 是否启用Debug模式
var Debug = false

func (s *httpServer) Run(addr, authToken string, bot *coolq.CQBot) {
	gin.SetMode(gin.ReleaseMode)
	s.engine = gin.New()
	s.bot = bot
	s.api = apiCaller{s.bot}
	s.engine.Use(func(c *gin.Context) {
		if c.Request.Method != "GET" && c.Request.Method != "POST" {
			log.Warnf("已拒绝客户端 %v 的请求: 方法错误", c.Request.RemoteAddr)
			c.Status(404)
			return
		}
		if c.Request.Method == "POST" && strings.Contains(c.Request.Header.Get("Content-Type"), "application/json") {
			d, err := c.GetRawData()
			if err != nil {
				log.Warnf("获取请求 %v 的Body时出现错误: %v", c.Request.RequestURI, err)
				c.Status(400)
				return
			}
			if !gjson.ValidBytes(d) {
				log.Warnf("已拒绝客户端 %v 的请求: 非法Json", c.Request.RemoteAddr)
				c.Status(400)
				return
			}
			c.Set("json_body", gjson.ParseBytes(d))
		}
		c.Next()
	})

	if authToken != "" {
		s.engine.Use(func(c *gin.Context) {
			auth := c.Request.Header.Get("Authorization")
			switch {
			case auth != "" && strings.SplitN(auth, " ", 2)[1] != authToken:
				c.AbortWithStatus(401)
				return
			case c.Query("access_token") != authToken:
				c.AbortWithStatus(401)
				return
			default:
				c.Next()
			}
		})
	}

	s.engine.Any("/:action", s.HandleActions)

	go func() {
		log.Infof("CQ HTTP 服务器已启动: %v", addr)
		s.HTTP = &http.Server{
			Addr:    addr,
			Handler: s.engine,
		}
		if err := s.HTTP.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error(err)
			log.Infof("HTTP 服务启动失败, 请检查端口是否被占用.")
			log.Warnf("将在五秒后退出.")
			time.Sleep(time.Second * 5)
			os.Exit(1)
		}
	}()
}

// NewHTTPClient 返回反向HTTP客户端
func NewHTTPClient() *HTTPClient {
	return &HTTPClient{}
}

// Run 运行反向HTTP服务
func (c *HTTPClient) Run(addr, secret string, timeout int32, bot *coolq.CQBot) {
	c.bot = bot
	c.secret = secret
	c.addr = addr
	c.timeout = timeout
	if c.timeout < 5 {
		c.timeout = 5
	}
	bot.OnEventPush(c.onBotPushEvent)
	log.Infof("HTTP POST上报器已启动: %v", addr)
}

func (c *HTTPClient) onBotPushEvent(m *bytes.Buffer) {
	var res string
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

func (s *httpServer) HandleActions(c *gin.Context) {
	global.RateLimit(context.Background())
	action := strings.ReplaceAll(c.Param("action"), "_async", "")
	log.Debugf("HTTPServer接收到API调用: %v", action)
	c.JSON(200, s.api.callAPI(action, httpContext{ctx: c}))
}

func (h httpContext) Get(k string) gjson.Result {
	c := h.ctx
	if q := c.Query(k); q != "" {
		return gjson.Result{Type: gjson.String, Str: q}
	}
	if c.Request.Method == "POST" {
		if h := c.Request.Header.Get("Content-Type"); h != "" {
			if strings.Contains(h, "application/x-www-form-urlencoded") {
				if p, ok := c.GetPostForm(k); ok {
					return gjson.Result{Type: gjson.String, Str: p}
				}
			}
			if strings.Contains(h, "application/json") {
				if obj, ok := c.Get("json_body"); ok {
					return obj.(gjson.Result).Get(k)
				}
			}
		}
	}
	return gjson.Result{Type: gjson.Null, Str: ""}
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
