package server

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"runtime/debug"
	"strings"

	"github.com/Mrs4s/MiraiGo/utils"
	log "github.com/sirupsen/logrus"

	"github.com/Mrs4s/go-cqhttp/coolq"
	"github.com/Mrs4s/go-cqhttp/global"
	"github.com/Mrs4s/go-cqhttp/global/config"
)

type lambdaClient struct {
	nextUrl     string
	responseURL string
	lambdaType  string

	client http.Client
	http   *httpServer
}

type lambdaResponse struct {
	IsBase64Encoded bool              `json:"isBase64Encoded"`
	StatusCode      int               `json:"statusCode"`
	Headers         map[string]string `json:"headers"`
	Body            string            `json:"body"`
}

type lambdaResponseWriter struct {
	statusCode int
	header     http.Header
	buf        *bytes.Buffer
}

func (l *lambdaResponseWriter) Header() http.Header {
	return l.header
}

func (l *lambdaResponseWriter) Write(data []byte) (int, error) {
	buffer := global.NewBuffer()
	defer global.PutBuffer(buffer)
	body := ""
	if l.buf != nil {
		body = utils.B2S(data)
	}
	header := make(map[string]string)
	for k, v := range l.header {
		header[k] = v[0]
	}
	json.NewEncoder(buffer).Encode(&lambdaResponse{
		IsBase64Encoded: false,
		StatusCode:      l.statusCode,
		Headers:         header,
		Body:            body,
	})

	r, _ := http.NewRequest("POST", cli.responseURL, buffer)
	do, err := cli.client.Do(r)
	if err != nil {
		return 0, err
	}
	_ = do.Body.Close()
	return len(data), nil
}

func (l *lambdaResponseWriter) WriteHeader(statusCode int) {
	l.statusCode = statusCode
}

var cli *lambdaClient

func init() {
	cli = &lambdaClient{
		lambdaType: "scf",
		client:     http.Client{Timeout: 0},
	}
	switch cli.lambdaType { // todo: aws
	case "scf": // tencent serverless function
		base := fmt.Sprintf("http://%s:%s/runtime/",
			os.Getenv("SCF_RUNTIME_API"),
			os.Getenv("SCF_RUNTIME_API_PORT"),
		)
		cli.nextUrl = base + "invocation/next"
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
		cli.nextUrl = base + "invocation/next"
		cli.responseURL = base + "invocation/response"
	}
}

// RunLambdaClient  type: [scf,aws]
func RunLambdaClient(bot *coolq.CQBot, conf *config.LambdaServer) {
	api := newAPICaller(bot)
	if conf.RateLimit.Enabled {
		api.use(rateLimit(conf.RateLimit.Frequency, conf.RateLimit.Bucket))
	}
	server := &httpServer{
		api:         api,
		accessToken: conf.AccessToken,
	}

	for {
		req := cli.next()
		func() {
			defer func() {
				if e := recover(); e != nil {
					log.Warnf("Lambda 出现不可恢复错误: %v\n%s", e, debug.Stack())
				}
			}()
			buffer := global.NewBuffer()
			defer global.PutBuffer(buffer)
			server.ServeHTTP(&lambdaResponseWriter{
				header: make(http.Header),
				buf:    buffer,
			}, req)
		}()
	}
}

type lambdaInvoke struct {
	Headers        map[string]string
	HttpMethod     string `json:"httpMethod"`
	Body           string `json:"body"`
	Path           string `json:"path"`
	QueryString    map[string]string
	RequestContext struct {
		Path string `json:"path"`
	} `json:"requestContext"`
}

func (c *lambdaClient) next() *http.Request {
	r, err := http.NewRequest(http.MethodGet, c.nextUrl, nil)
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
	var req = new(http.Request)
	var invoke = new(lambdaInvoke)
	_ = json.NewDecoder(resp.Body).Decode(invoke)

	req.Method = invoke.HttpMethod
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
	return req
}
