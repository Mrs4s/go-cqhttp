package server

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Mrs4s/MiraiGo/utils"
	"github.com/RomiChan/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"gopkg.in/yaml.v3"

	"github.com/Mrs4s/go-cqhttp/coolq"
	"github.com/Mrs4s/go-cqhttp/global"
	"github.com/Mrs4s/go-cqhttp/modules/api"
	"github.com/Mrs4s/go-cqhttp/modules/config"
	"github.com/Mrs4s/go-cqhttp/modules/filter"
	"github.com/Mrs4s/go-cqhttp/pkg/onebot"
)

type webSocketServer struct {
	bot  *coolq.CQBot
	conf *WebsocketServer

	mu        sync.Mutex
	eventConn []*wsConn

	token     string
	handshake string
	filter    string
}

// websocketClient WebSocket客户端实例
type websocketClient struct {
	bot       *coolq.CQBot
	mu        sync.Mutex
	universal *wsConn
	event     *wsConn

	token             string
	filter            string
	reconnectInterval time.Duration
	limiter           api.Handler
}

type wsConn struct {
	mu        sync.Mutex
	conn      *websocket.Conn
	apiCaller *api.Caller
}

func (c *wsConn) WriteText(b []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	_ = c.conn.SetWriteDeadline(time.Now().Add(time.Second * 15))
	return c.conn.WriteMessage(websocket.TextMessage, b)
}

func (c *wsConn) Close() error {
	return c.conn.Close()
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

const wsDefault = `  # 正向WS设置
  - ws:
      # 正向WS服务器监听地址
      address: 0.0.0.0:8080
      middlewares:
        <<: *default # 引用默认中间件
`

const wsReverseDefault = `  # 反向WS设置
  - ws-reverse:
      # 反向WS Universal 地址
      # 注意 设置了此项地址后下面两项将会被忽略
      universal: ws://your_websocket_universal.server
      # 反向WS API 地址
      api: ws://your_websocket_api.server
      # 反向WS Event 地址
      event: ws://your_websocket_event.server
      # 重连间隔 单位毫秒
      reconnect-interval: 3000
      middlewares:
        <<: *default # 引用默认中间件
`

// WebsocketServer 正向WS相关配置
type WebsocketServer struct {
	Disabled bool   `yaml:"disabled"`
	Address  string `yaml:"address"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`

	MiddleWares `yaml:"middlewares"`
}

// WebsocketReverse 反向WS相关配置
type WebsocketReverse struct {
	Disabled          bool   `yaml:"disabled"`
	Universal         string `yaml:"universal"`
	API               string `yaml:"api"`
	Event             string `yaml:"event"`
	ReconnectInterval int    `yaml:"reconnect-interval"`

	MiddleWares `yaml:"middlewares"`
}

func init() {
	config.AddServer(&config.Server{
		Brief:   "正向 Websocket 通信",
		Default: wsDefault,
	})
	config.AddServer(&config.Server{
		Brief:   "反向 Websocket 通信",
		Default: wsReverseDefault,
	})
}

// runWSServer 运行一个正向WS server
func runWSServer(b *coolq.CQBot, node yaml.Node) {
	var conf WebsocketServer
	switch err := node.Decode(&conf); {
	case err != nil:
		log.Warn("读取正向Websocket配置失败 :", err)
		fallthrough
	case conf.Disabled:
		return
	}

	network, address := "tcp", conf.Address
	if conf.Address == "" && (conf.Host != "" || conf.Port != 0) {
		log.Warn("正向 Websocket 使用了过时的配置格式，请更新配置文件")
		address = fmt.Sprintf("%s:%d", conf.Host, conf.Port)
	} else {
		uri, err := url.Parse(conf.Address)
		if err == nil && uri.Scheme != "" {
			network = uri.Scheme
			address = uri.Host + uri.Path
		}
	}
	s := &webSocketServer{
		bot:    b,
		conf:   &conf,
		token:  conf.AccessToken,
		filter: conf.Filter,
	}
	filter.Add(s.filter)
	s.handshake = fmt.Sprintf(`{"_post_method":2,"meta_event_type":"lifecycle","post_type":"meta_event","self_id":%d,"sub_type":"connect","time":%d}`,
		b.Client.Uin, time.Now().Unix())
	b.OnEventPush(s.onBotPushEvent)
	mux := http.ServeMux{}
	mux.HandleFunc("/event", s.event)
	mux.HandleFunc("/api", s.api)
	mux.HandleFunc("/", s.any)
	listener, err := net.Listen(network, address)
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("CQ WebSocket 服务器已启动: %v", listener.Addr())
	log.Fatal(http.Serve(listener, &mux))
}

// runWSClient 运行一个反向向WS client
func runWSClient(b *coolq.CQBot, node yaml.Node) {
	var conf WebsocketReverse
	switch err := node.Decode(&conf); {
	case err != nil:
		log.Warn("读取反向Websocket配置失败 :", err)
		fallthrough
	case conf.Disabled:
		return
	}

	c := &websocketClient{
		bot:    b,
		token:  conf.AccessToken,
		filter: conf.Filter,
	}
	filter.Add(c.filter)

	if conf.ReconnectInterval != 0 {
		c.reconnectInterval = time.Duration(conf.ReconnectInterval) * time.Millisecond
	} else {
		c.reconnectInterval = time.Second * 5
	}

	if conf.RateLimit.Enabled {
		c.limiter = rateLimit(conf.RateLimit.Frequency, conf.RateLimit.Bucket)
	}

	if conf.Universal != "" {
		c.connect("Universal", conf.Universal, &c.universal)
		c.bot.OnEventPush(c.onBotPushEvent("Universal", conf.Universal, &c.universal))
		return // 连接到 Universal 后， 不再连接其他
	}
	if conf.API != "" {
		c.connect("API", conf.API, nil)
	}
	if conf.Event != "" {
		c.connect("Event", conf.Event, &c.event)
		c.bot.OnEventPush(c.onBotPushEvent("Event", conf.Event, &c.event))
	}
}

func resolveURI(addr string) (network, address string) {
	network, address = "tcp", addr
	uri, err := url.Parse(addr)
	if err == nil && uri.Scheme != "" {
		scheme, ext, _ := strings.Cut(uri.Scheme, "+")
		if ext != "" {
			network = ext
			uri.Scheme = scheme // remove `+unix`/`+tcp4`
			if ext == "unix" {
				uri.Host, uri.Path, _ = strings.Cut(uri.Path, ":")
				uri.Host = base64.StdEncoding.EncodeToString([]byte(uri.Host))
			}
			address = uri.String()
		}
	}
	return
}

func (c *websocketClient) connect(typ, addr string, conptr **wsConn) {
	log.Infof("开始尝试连接到反向WebSocket %s服务器: %v", typ, addr)
	header := http.Header{
		"X-Client-Role": []string{typ},
		"X-Self-ID":     []string{strconv.FormatInt(c.bot.Client.Uin, 10)},
		"User-Agent":    []string{"CQHttp/4.15.0"},
	}
	if c.token != "" {
		header["Authorization"] = []string{"Token " + c.token}
	}

	network, address := resolveURI(addr)
	dialer := websocket.Dialer{
		NetDial: func(_, addr string) (net.Conn, error) {
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
			return net.Dial(network, addr) // support unix socket transport
		},
	}

	conn, _, err := dialer.Dial(address, header) // nolint
	if err != nil {
		log.Warnf("连接到反向WebSocket %s服务器 %v 时出现错误: %v", typ, addr, err)
		if c.reconnectInterval != 0 {
			time.Sleep(c.reconnectInterval)
			c.connect(typ, addr, conptr)
		}
		return
	}

	switch typ {
	case "Event", "Universal":
		handshake := fmt.Sprintf(`{"meta_event_type":"lifecycle","post_type":"meta_event","self_id":%d,"sub_type":"connect","time":%d}`, c.bot.Client.Uin, time.Now().Unix())
		err = conn.WriteMessage(websocket.TextMessage, []byte(handshake))
		if err != nil {
			log.Warnf("反向WebSocket 握手时出现错误: %v", err)
		}
	}

	log.Infof("已连接到反向WebSocket %s服务器 %v", typ, addr)

	var wrappedConn *wsConn
	if conptr != nil && *conptr != nil {
		wrappedConn = *conptr
	} else {
		wrappedConn = new(wsConn)
		if conptr != nil {
			*conptr = wrappedConn
		}
	}

	wrappedConn.conn = conn
	wrappedConn.apiCaller = api.NewCaller(c.bot)
	if c.limiter != nil {
		wrappedConn.apiCaller.Use(c.limiter)
	}

	if typ != "Event" {
		go c.listenAPI(typ, addr, wrappedConn)
	}
}

func (c *websocketClient) listenAPI(typ, url string, conn *wsConn) {
	defer func() { _ = conn.Close() }()
	for {
		buffer := global.NewBuffer()
		t, reader, err := conn.conn.NextReader()
		if err != nil {
			log.Warnf("监听反向WS %s时出现错误: %v", typ, err)
			break
		}
		_, err = buffer.ReadFrom(reader)
		if err != nil {
			log.Warnf("监听反向WS %s时出现错误: %v", typ, err)
			break
		}
		if t == websocket.TextMessage {
			go func(buffer *bytes.Buffer) {
				defer global.PutBuffer(buffer)
				conn.handleRequest(c.bot, buffer.Bytes())
			}(buffer)
		} else {
			global.PutBuffer(buffer)
		}
	}
	if c.reconnectInterval != 0 {
		time.Sleep(c.reconnectInterval)
		if typ == "API" { // Universal 不重连，避免多次重连
			go c.connect(typ, url, nil)
		}
	}
}

func (c *websocketClient) onBotPushEvent(typ, url string, conn **wsConn) func(e *coolq.Event) {
	return func(e *coolq.Event) {
		c.mu.Lock()
		defer c.mu.Unlock()

		flt := filter.Find(c.filter)
		if flt != nil && !flt.Eval(gjson.Parse(e.JSONString())) {
			log.Debugf("上报Event %s 到 WS服务器 时被过滤.", e.JSONBytes())
			return
		}

		log.Debugf("向反向WS %s服务器推送Event: %s", typ, e.JSONBytes())
		if err := (*conn).WriteText(e.JSONBytes()); err != nil {
			log.Warnf("向反向WS %s服务器推送 Event 时出现错误: %v", typ, err)
			_ = (*conn).Close()
			if c.reconnectInterval != 0 {
				time.Sleep(c.reconnectInterval)
				c.connect(typ, url, conn)
			}
		}
	}
}

func (s *webSocketServer) event(w http.ResponseWriter, r *http.Request) {
	status := checkAuth(r, s.token)
	if status != http.StatusOK {
		log.Warnf("已拒绝 %v 的 WebSocket 请求: Token鉴权失败(code:%d)", r.RemoteAddr, status)
		w.WriteHeader(status)
		return
	}

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Warnf("处理 WebSocket 请求时出现错误: %v", err)
		return
	}

	err = c.WriteMessage(websocket.TextMessage, []byte(s.handshake))
	if err != nil {
		log.Warnf("WebSocket 握手时出现错误: %v", err)
		_ = c.Close()
		return
	}

	log.Infof("接受 WebSocket 连接: %v (/event)", r.RemoteAddr)
	conn := &wsConn{conn: c, apiCaller: api.NewCaller(s.bot)}
	s.mu.Lock()
	s.eventConn = append(s.eventConn, conn)
	s.mu.Unlock()
}

func (s *webSocketServer) api(w http.ResponseWriter, r *http.Request) {
	status := checkAuth(r, s.token)
	if status != http.StatusOK {
		log.Warnf("已拒绝 %v 的 WebSocket 请求: Token鉴权失败(code:%d)", r.RemoteAddr, status)
		w.WriteHeader(status)
		return
	}

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Warnf("处理 WebSocket 请求时出现错误: %v", err)
		return
	}

	log.Infof("接受 WebSocket 连接: %v (/api)", r.RemoteAddr)
	conn := &wsConn{conn: c, apiCaller: api.NewCaller(s.bot)}
	if s.conf.RateLimit.Enabled {
		conn.apiCaller.Use(rateLimit(s.conf.RateLimit.Frequency, s.conf.RateLimit.Bucket))
	}
	s.listenAPI(conn)
}

func (s *webSocketServer) any(w http.ResponseWriter, r *http.Request) {
	status := checkAuth(r, s.token)
	if status != http.StatusOK {
		log.Warnf("已拒绝 %v 的 WebSocket 请求: Token鉴权失败(code:%d)", r.RemoteAddr, status)
		w.WriteHeader(status)
		return
	}

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Warnf("处理 WebSocket 请求时出现错误: %v", err)
		return
	}

	err = c.WriteMessage(websocket.TextMessage, []byte(s.handshake))
	if err != nil {
		log.Warnf("WebSocket 握手时出现错误: %v", err)
		_ = c.Close()
		return
	}

	log.Infof("接受 WebSocket 连接: %v (/)", r.RemoteAddr)
	conn := &wsConn{conn: c, apiCaller: api.NewCaller(s.bot)}
	if s.conf.RateLimit.Enabled {
		conn.apiCaller.Use(rateLimit(s.conf.RateLimit.Frequency, s.conf.RateLimit.Bucket))
	}
	s.mu.Lock()
	s.eventConn = append(s.eventConn, conn)
	s.mu.Unlock()
	s.listenAPI(conn)
}

func (s *webSocketServer) listenAPI(c *wsConn) {
	defer func() { _ = c.Close() }()
	for {
		buffer := global.NewBuffer()
		t, reader, err := c.conn.NextReader()
		if err != nil {
			break
		}
		_, err = buffer.ReadFrom(reader)
		if err != nil {
			break
		}

		if t == websocket.TextMessage {
			go func(buffer *bytes.Buffer) {
				defer global.PutBuffer(buffer)
				c.handleRequest(s.bot, buffer.Bytes())
			}(buffer)
		} else {
			global.PutBuffer(buffer)
		}
	}
}

func (c *wsConn) handleRequest(_ *coolq.CQBot, payload []byte) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("处置WS命令时发生无法恢复的异常：%v\n%s", err, debug.Stack())
			_ = c.Close()
		}
	}()

	j := gjson.Parse(utils.B2S(payload))
	t := strings.TrimSuffix(j.Get("action").Str, "_async")
	params := j.Get("params")
	log.Debugf("WS接收到API调用: %v 参数: %v", t, params.Raw)
	ret := c.apiCaller.Call(t, onebot.V11, params)
	if j.Get("echo").Exists() {
		ret["echo"] = j.Get("echo").Value()
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	_ = c.conn.SetWriteDeadline(time.Now().Add(time.Second * 15))
	writer, err := c.conn.NextWriter(websocket.TextMessage)
	if err != nil {
		log.Errorf("无法响应API调用(连接已断开?): %v", err)
		return
	}
	_ = json.NewEncoder(writer).Encode(ret)
	_ = writer.Close()
}

func (s *webSocketServer) onBotPushEvent(e *coolq.Event) {
	flt := filter.Find(s.filter)
	if flt != nil && !flt.Eval(gjson.Parse(e.JSONString())) {
		log.Debugf("上报Event %s 到 WS客户端 时被过滤.", e.JSONBytes())
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	j := 0
	for i := 0; i < len(s.eventConn); i++ {
		conn := s.eventConn[i]
		log.Debugf("向WS客户端推送Event: %s", e.JSONBytes())
		if err := conn.WriteText(e.JSONBytes()); err != nil {
			_ = conn.Close()
			conn = nil
			continue
		}
		if i != j {
			// i != j means that some connection has been closed.
			// use an in-place removal to avoid copying.
			s.eventConn[j] = conn
		}
		j++
	}
	s.eventConn = s.eventConn[:j]
}
