package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"runtime/debug"
	"strings"

	"github.com/Mrs4s/MiraiGo/utils"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"github.com/Mrs4s/go-cqhttp/coolq"
	"github.com/Mrs4s/go-cqhttp/global"
	api2 "github.com/Mrs4s/go-cqhttp/modules/api"
	"github.com/Mrs4s/go-cqhttp/modules/config"
)

type lambdaClient struct {
	nextURL     string
	responseURL string
	lambdaType  string

	client http.Client
}

type lambdaResponse struct {
	IsBase64Encoded bool              `json:"isBase64Encoded"`
	StatusCode      int               `json:"statusCode"`
	Headers         map[string]string `json:"headers"`
	Body            string            `json:"body"`
}

type lambdaResponseWriter struct {
	statusCode int
	buf        bytes.Buffer
	header     http.Header
}

func (l *lambdaResponseWriter) Write(p []byte) (n int, err error) {
	return l.buf.Write(p)
}

func (l *lambdaResponseWriter) Header() http.Header {
	return l.header
}

func (l *lambdaResponseWriter) flush() error {
	buffer := global.NewBuffer()
	defer global.PutBuffer(buffer)
	body := utils.B2S(l.buf.Bytes())
	header := make(map[string]string, len(l.header))
	for k, v := range l.header {
		header[k] = v[0]
	}
	_ = json.NewEncoder(buffer).Encode(&lambdaResponse{
		IsBase64Encoded: false,
		StatusCode:      l.statusCode,
		Headers:         header,
		Body:            body,
	})

	r, _ := http.NewRequest(http.MethodPost, cli.responseURL, buffer)
	do, err := cli.client.Do(r)
	if err != nil {
		return err
	}
	return do.Body.Close()
}

func (l *lambdaResponseWriter) WriteHeader(statusCode int) {
	l.statusCode = statusCode
}

var cli *lambdaClient

// runLambda  type: [scf,aws]
func runLambda(bot *coolq.CQBot, node yaml.Node) {
	var conf LambdaServer
	switch err := node.Decode(&conf); {
	case err != nil:
		log.Warn("读取lambda配置失败 :", err)
		fallthrough
	case conf.Disabled:
		return
	}

	cli = &lambdaClient{
		lambdaType: conf.Type,
		client:     http.Client{Timeout: 0},
	}
	switch cli.lambdaType { // todo: aws
	case "scf": // tencent serverless function
		base := fmt.Sprintf("http://%s:%s/runtime/",
			os.Getenv("SCF_RUNTIME_API"),
			os.Getenv("SCF_RUNTIME_API_PORT"))
		cli.nextURL = base + "invocation/next"
		cli.responseURL = base + "invocation/response"
		post, err := http.Post(base+"init/ready", "", nil)
		if err != nil {
			log.Warnf("lambda 初始化失败: %v", err)
			return
		}
		_ = post.Body.Close()
	case "aws": // aws lambda
		const apiVersion = "2018-06-01"
		base := fmt.Sprintf("http://%s/%s/runtime/", os.Getenv("AWS_LAMBDA_RUNTIME_API"), apiVersion)
		cli.nextURL = base + "invocation/next"
		cli.responseURL = base + "invocation/response"
	default:
		log.Fatal("unknown lambda type:", conf.Type)
	}

	api := api2.NewCaller(bot)
	if conf.RateLimit.Enabled {
		api.Use(rateLimit(conf.RateLimit.Frequency, conf.RateLimit.Bucket))
	}
	server := &httpServer{
		api:         api,
		accessToken: conf.AccessToken,
	}

	for {
		req := cli.next()
		writer := lambdaResponseWriter{statusCode: 200, header: make(http.Header)}
		func() {
			defer func() {
				if e := recover(); e != nil {
					log.Warnf("Lambda 出现不可恢复错误: %v\n%s", e, debug.Stack())
				}
			}()
			if req != nil {
				server.ServeHTTP(&writer, req)
			}
		}()
		if err := writer.flush(); err != nil {
			log.Warnf("Lambda 发送响应失败: %v", err)
		}
	}
}

type lambdaInvoke struct {
	Headers        map[string]string
	HTTPMethod     string `json:"httpMethod"`
	Body           string `json:"body"`
	Path           string `json:"path"`
	QueryString    map[string]string
	RequestContext struct {
		Path string `json:"path"`
	} `json:"requestContext"`
}

const lambdaDefault = `  # LambdaServer 配置
  - lambda:
      type: scf # scf: 腾讯云函数 aws: aws Lambda
      middlewares:
        <<: *default # 引用默认中间件
`

// LambdaServer 云函数配置
type LambdaServer struct {
	Disabled bool   `yaml:"disabled"`
	Type     string `yaml:"type"`

	MiddleWares `yaml:"middlewares"`
}

func init() {
	config.AddServer(&config.Server{
		Brief:   "云函数服务",
		Default: lambdaDefault,
	})
}

func (c *lambdaClient) next() *http.Request {
	r, err := http.NewRequest(http.MethodGet, c.nextURL, nil)
	if err != nil {
		return nil
	}
	resp, err := c.client.Do(r)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	var req http.Request
	var invoke lambdaInvoke
	_ = json.NewDecoder(resp.Body).Decode(&invoke)
	if invoke.HTTPMethod == "" { // 不是 api 网关
		return nil
	}

	req.Method = invoke.HTTPMethod
	req.Body = io.NopCloser(strings.NewReader(invoke.Body))
	req.Header = make(map[string][]string)
	for k, v := range invoke.Headers {
		req.Header.Set(k, v)
	}
	req.URL = new(url.URL)
	req.URL.Path = strings.TrimPrefix(invoke.Path, invoke.RequestContext.Path)
	// todo: avoid encoding
	query := make(url.Values)
	for k, v := range invoke.QueryString {
		query[k] = []string{v}
	}
	req.URL.RawQuery = query.Encode()
	return &req
}
