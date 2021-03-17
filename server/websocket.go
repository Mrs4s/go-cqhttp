package server

import (
	"context"
	"fmt"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Mrs4s/go-cqhttp/coolq"
	"github.com/Mrs4s/go-cqhttp/global"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

type webSocketServer struct {
	bot            *coolq.CQBot
	token          string
	eventConn      []*webSocketConn
	eventConnMutex sync.Mutex
	handshake      string
}

// WebSocketClient WebSocket客户端实例
type WebSocketClient struct {
	conf  *global.GoCQReverseWebSocketConfig
	token string
	bot   *coolq.CQBot

	universalConn *webSocketConn
	eventConn     *webSocketConn
}

type webSocketConn struct {
	*websocket.Conn
	sync.Mutex
	apiCaller apiCaller
}

// WebSocketServer 初始化一个WebSocketServer实例
var WebSocketServer = &webSocketServer{}
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (s *webSocketServer) Run(addr, authToken string, b *coolq.CQBot) {
	s.token = authToken
	s.bot = b
	s.handshake = fmt.Sprintf(`{"_post_method":2,"meta_event_type":"lifecycle","post_type":"meta_event","self_id":%d,"sub_type":"connect","time":%d}`,
		s.bot.Client.Uin, time.Now().Unix())
	b.OnEventPush(s.onBotPushEvent)
	http.HandleFunc("/event", s.event)
	http.HandleFunc("/api", s.api)
	http.HandleFunc("/", s.any)
	go func() {
		log.Infof("CQ WebSocket 服务器已启动: %v", addr)
		log.Fatal(http.ListenAndServe(addr, nil))
	}()
}

// NewWebSocketClient 初始化一个NWebSocket客户端
func NewWebSocketClient(conf *global.GoCQReverseWebSocketConfig, authToken string, b *coolq.CQBot) *WebSocketClient {
	return &WebSocketClient{conf: conf, token: authToken, bot: b}
}

// Run 运行实例
func (c *WebSocketClient) Run() {
	if !c.conf.Enabled {
		return
	}
	if c.conf.ReverseURL != "" {
		c.connectUniversal()
	} else {
		if c.conf.ReverseAPIURL != "" {
			c.connectAPI()
		}
		if c.conf.ReverseEventURL != "" {
			c.connectEvent()
		}
	}
	c.bot.OnEventPush(c.onBotPushEvent)
}

func (c *WebSocketClient) connectAPI() {
	log.Infof("开始尝试连接到反向WebSocket API服务器: %v", c.conf.ReverseAPIURL)
	header := http.Header{
		"X-Client-Role": []string{"API"},
		"X-Self-ID":     []string{strconv.FormatInt(c.bot.Client.Uin, 10)},
		"User-Agent":    []string{"CQHttp/4.15.0"},
	}
	if c.token != "" {
		header["Authorization"] = []string{"Token " + c.token}
	}
	conn, _, err := websocket.DefaultDialer.Dial(c.conf.ReverseAPIURL, header) // nolint
	if err != nil {
		log.Warnf("连接到反向WebSocket API服务器 %v 时出现错误: %v", c.conf.ReverseAPIURL, err)
		if c.conf.ReverseReconnectInterval != 0 {
			time.Sleep(time.Millisecond * time.Duration(c.conf.ReverseReconnectInterval))
			c.connectAPI()
		}
		return
	}
	log.Infof("已连接到反向WebSocket API服务器 %v", c.conf.ReverseAPIURL)
	wrappedConn := &webSocketConn{Conn: conn, apiCaller: apiCaller{c.bot}}
	go c.listenAPI(wrappedConn, false)
}

func (c *WebSocketClient) connectEvent() {
	log.Infof("开始尝试连接到反向WebSocket Event服务器: %v", c.conf.ReverseEventURL)
	header := http.Header{
		"X-Client-Role": []string{"Event"},
		"X-Self-ID":     []string{strconv.FormatInt(c.bot.Client.Uin, 10)},
		"User-Agent":    []string{"CQHttp/4.15.0"},
	}
	if c.token != "" {
		header["Authorization"] = []string{"Token " + c.token}
	}
	conn, _, err := websocket.DefaultDialer.Dial(c.conf.ReverseEventURL, header) // nolint
	if err != nil {
		log.Warnf("连接到反向WebSocket Event服务器 %v 时出现错误: %v", c.conf.ReverseEventURL, err)
		if c.conf.ReverseReconnectInterval != 0 {
			time.Sleep(time.Millisecond * time.Duration(c.conf.ReverseReconnectInterval))
			c.connectEvent()
		}
		return
	}

	handshake := fmt.Sprintf(`{"meta_event_type":"lifecycle","post_type":"meta_event","self_id":%d,"sub_type":"connect","time":%d}`,
		c.bot.Client.Uin, time.Now().Unix())
	err = conn.WriteMessage(websocket.TextMessage, []byte(handshake))
	if err != nil {
		log.Warnf("反向WebSocket 握手时出现错误: %v", err)
	}

	log.Infof("已连接到反向WebSocket Event服务器 %v", c.conf.ReverseEventURL)
	c.eventConn = &webSocketConn{Conn: conn, apiCaller: apiCaller{c.bot}}
}

func (c *WebSocketClient) connectUniversal() {
	log.Infof("开始尝试连接到反向WebSocket Universal服务器: %v", c.conf.ReverseURL)
	header := http.Header{
		"X-Client-Role": []string{"Universal"},
		"X-Self-ID":     []string{strconv.FormatInt(c.bot.Client.Uin, 10)},
		"User-Agent":    []string{"CQHttp/4.15.0"},
	}
	if c.token != "" {
		header["Authorization"] = []string{"Token " + c.token}
	}
	conn, _, err := websocket.DefaultDialer.Dial(c.conf.ReverseURL, header) // nolint
	if err != nil {
		log.Warnf("连接到反向WebSocket Universal服务器 %v 时出现错误: %v", c.conf.ReverseURL, err)
		if c.conf.ReverseReconnectInterval != 0 {
			time.Sleep(time.Millisecond * time.Duration(c.conf.ReverseReconnectInterval))
			c.connectUniversal()
		}
		return
	}
	handshake := fmt.Sprintf(`{"meta_event_type":"lifecycle","post_type":"meta_event","self_id":%d,"sub_type":"connect","time":%d}`,
		c.bot.Client.Uin, time.Now().Unix())
	err = conn.WriteMessage(websocket.TextMessage, []byte(handshake))
	if err != nil {
		log.Warnf("反向WebSocket 握手时出现错误: %v", err)
	}

	wrappedConn := &webSocketConn{Conn: conn, apiCaller: apiCaller{c.bot}}
	go c.listenAPI(wrappedConn, true)
	c.universalConn = wrappedConn
}

func (c *WebSocketClient) listenAPI(conn *webSocketConn, u bool) {
	defer conn.Close()
	for {
		_, buf, err := conn.ReadMessage()
		if err != nil {
			log.Warnf("监听反向WS API时出现错误: %v", err)
			break
		}

		go conn.handleRequest(c.bot, buf)
	}
	if c.conf.ReverseReconnectInterval != 0 {
		time.Sleep(time.Millisecond * time.Duration(c.conf.ReverseReconnectInterval))
		if !u {
			go c.connectAPI()
		}
	}
}

func (c *WebSocketClient) onBotPushEvent(m coolq.MSG) {
	if c.eventConn != nil {
		log.Debugf("向WS服务器 %v 推送Event: %v", c.eventConn.RemoteAddr().String(), m.ToJSON())
		conn := c.eventConn
		conn.Lock()
		defer conn.Unlock()
		_ = c.eventConn.SetWriteDeadline(time.Now().Add(time.Second * 15))
		if err := c.eventConn.WriteJSON(m); err != nil {
			log.Warnf("向WS服务器 %v 推送Event时出现错误: %v", c.eventConn.RemoteAddr().String(), err)
			_ = c.eventConn.Close()
			if c.conf.ReverseReconnectInterval != 0 {
				time.Sleep(time.Millisecond * time.Duration(c.conf.ReverseReconnectInterval))
				c.connectEvent()
			}
		}
	}
	if c.universalConn != nil {
		log.Debugf("向WS服务器 %v 推送Event: %v", c.universalConn.RemoteAddr().String(), m.ToJSON())
		conn := c.universalConn
		conn.Lock()
		defer conn.Unlock()
		_ = c.universalConn.SetWriteDeadline(time.Now().Add(time.Second * 15))
		if err := c.universalConn.WriteJSON(m); err != nil {
			log.Warnf("向WS服务器 %v 推送Event时出现错误: %v", c.universalConn.RemoteAddr().String(), err)
			_ = c.universalConn.Close()
			if c.conf.ReverseReconnectInterval != 0 {
				time.Sleep(time.Millisecond * time.Duration(c.conf.ReverseReconnectInterval))
				c.connectUniversal()
			}
		}
	}
}

func (s *webSocketServer) event(w http.ResponseWriter, r *http.Request) {
	if s.token != "" {
		if auth := r.URL.Query().Get("access_token"); auth != s.token {
			if auth := strings.SplitN(r.Header.Get("Authorization"), " ", 2); len(auth) != 2 || auth[1] != s.token {
				log.Warnf("已拒绝 %v 的 WebSocket 请求: Token鉴权失败", r.RemoteAddr)
				w.WriteHeader(401)
				return
			}
		}
	}
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Warnf("处理 WebSocket 请求时出现错误: %v", err)
		return
	}
	err = c.WriteMessage(websocket.TextMessage, []byte(s.handshake))
	if err != nil {
		log.Warnf("WebSocket 握手时出现错误: %v", err)
		c.Close()
		return
	}

	log.Infof("接受 WebSocket 连接: %v (/event)", r.RemoteAddr)

	conn := &webSocketConn{Conn: c, apiCaller: apiCaller{s.bot}}

	s.eventConnMutex.Lock()
	s.eventConn = append(s.eventConn, conn)
	s.eventConnMutex.Unlock()
}

func (s *webSocketServer) api(w http.ResponseWriter, r *http.Request) {
	if s.token != "" {
		if auth := r.URL.Query().Get("access_token"); auth != s.token {
			if auth := strings.SplitN(r.Header.Get("Authorization"), " ", 2); len(auth) != 2 || auth[1] != s.token {
				log.Warnf("已拒绝 %v 的 WebSocket 请求: Token鉴权失败", r.RemoteAddr)
				w.WriteHeader(401)
				return
			}
		}
	}
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Warnf("处理 WebSocket 请求时出现错误: %v", err)
		return
	}
	log.Infof("接受 WebSocket 连接: %v (/api)", r.RemoteAddr)
	conn := &webSocketConn{Conn: c, apiCaller: apiCaller{s.bot}}
	go s.listenAPI(conn)
}

func (s *webSocketServer) any(w http.ResponseWriter, r *http.Request) {
	if s.token != "" {
		if auth := r.URL.Query().Get("access_token"); auth != s.token {
			if auth := strings.SplitN(r.Header.Get("Authorization"), " ", 2); len(auth) != 2 || auth[1] != s.token {
				log.Warnf("已拒绝 %v 的 WebSocket 请求: Token鉴权失败", r.RemoteAddr)
				w.WriteHeader(401)
				return
			}
		}
	}
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Warnf("处理 WebSocket 请求时出现错误: %v", err)
		return
	}
	err = c.WriteMessage(websocket.TextMessage, []byte(s.handshake))
	if err != nil {
		log.Warnf("WebSocket 握手时出现错误: %v", err)
		c.Close()
		return
	}
	log.Infof("接受 WebSocket 连接: %v (/)", r.RemoteAddr)
	conn := &webSocketConn{Conn: c, apiCaller: apiCaller{s.bot}}
	s.eventConn = append(s.eventConn, conn)
	s.listenAPI(conn)
}

func (s *webSocketServer) listenAPI(c *webSocketConn) {
	defer c.Close()
	for {
		t, payload, err := c.ReadMessage()
		if err != nil {
			break
		}

		if t == websocket.TextMessage {
			go c.handleRequest(s.bot, payload)
		}
	}
}

func (c *webSocketConn) handleRequest(_ *coolq.CQBot, payload []byte) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("处置WS命令时发生无法恢复的异常：%v\n%s", err, debug.Stack())
			c.Close()
		}
	}()
	global.RateLimit(context.Background())
	j := gjson.ParseBytes(payload)
	t := strings.ReplaceAll(j.Get("action").Str, "_async", "")
	log.Debugf("WS接收到API调用: %v 参数: %v", t, j.Get("params").Raw)
	ret := c.apiCaller.callAPI(t, j.Get("params"))
	if j.Get("echo").Exists() {
		ret["echo"] = j.Get("echo").Value()
	}
	c.Lock()
	defer c.Unlock()
	_ = c.WriteJSON(ret)
}

func (s *webSocketServer) onBotPushEvent(m coolq.MSG) {
	s.eventConnMutex.Lock()
	defer s.eventConnMutex.Unlock()
	for i, l := 0, len(s.eventConn); i < l; i++ {
		conn := s.eventConn[i]
		log.Debugf("向WS客户端 %v 推送Event: %v", conn.RemoteAddr().String(), m.ToJSON())
		conn.Lock()
		if err := conn.WriteMessage(websocket.TextMessage, []byte(m.ToJSON())); err != nil {
			_ = conn.Close()
			next := i + 1
			if next >= l {
				next = l - 1
			}
			s.eventConn[i], s.eventConn[next] = s.eventConn[next], s.eventConn[i]
			s.eventConn = append(s.eventConn[:next], s.eventConn[next+1:]...)
			i--
			l--
			conn = nil
			continue
		}
		conn.Unlock()
	}
}
