package server

import (
	"bytes"
	"fmt"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Mrs4s/go-cqhttp/coolq"
	"github.com/Mrs4s/go-cqhttp/global"
	"github.com/Mrs4s/go-cqhttp/global/config"

	"github.com/Mrs4s/MiraiGo/utils"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

type webSocketServer struct {
	bot  *coolq.CQBot
	conf *config.WebsocketServer

	eventConn      []*webSocketConn
	eventConnMutex sync.Mutex
	token          string
	handshake      string
	filter         string
}

// websocketClient WebSocket客户端实例
type websocketClient struct {
	bot  *coolq.CQBot
	conf *config.WebsocketReverse

	universalConn *webSocketConn
	eventConn     *webSocketConn
	token         string
	filter        string
}

type webSocketConn struct {
	*websocket.Conn
	sync.Mutex
	apiCaller *apiCaller
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// RunWebSocketServer 运行一个正向WS server
func RunWebSocketServer(b *coolq.CQBot, conf *config.WebsocketServer) {
	if conf.Disabled {
		return
	}
	s := &webSocketServer{
		bot:    b,
		conf:   conf,
		token:  conf.AccessToken,
		filter: conf.Filter,
	}
	addFilter(s.filter)
	addr := fmt.Sprintf("%s:%d", conf.Host, conf.Port)
	s.handshake = fmt.Sprintf(`{"_post_method":2,"meta_event_type":"lifecycle","post_type":"meta_event","self_id":%d,"sub_type":"connect","time":%d}`,
		b.Client.Uin, time.Now().Unix())
	b.OnEventPush(s.onBotPushEvent)
	mux := http.ServeMux{}
	mux.HandleFunc("/event", s.event)
	mux.HandleFunc("/api", s.api)
	mux.HandleFunc("/", s.any)
	go func() {
		log.Infof("CQ WebSocket 服务器已启动: %v", addr)
		log.Fatal(http.ListenAndServe(addr, &mux))
	}()
}

// RunWebSocketClient 运行一个正向WS client
func RunWebSocketClient(b *coolq.CQBot, conf *config.WebsocketReverse) {
	if conf.Disabled {
		return
	}
	c := &websocketClient{
		bot:    b,
		conf:   conf,
		token:  conf.AccessToken,
		filter: conf.Filter,
	}
	addFilter(c.filter)
	if c.conf.Universal != "" {
		c.connectUniversal()
	} else {
		if c.conf.API != "" {
			c.connectAPI()
		}
		if c.conf.Event != "" {
			c.connectEvent()
		}
	}
	c.bot.OnEventPush(c.onBotPushEvent)
}

func (c *websocketClient) connectAPI() {
	log.Infof("开始尝试连接到反向WebSocket API服务器: %v", c.conf.API)
	header := http.Header{
		"X-Client-Role": []string{"API"},
		"X-Self-ID":     []string{strconv.FormatInt(c.bot.Client.Uin, 10)},
		"User-Agent":    []string{"CQHttp/4.15.0"},
	}
	if c.token != "" {
		header["Authorization"] = []string{"Token " + c.token}
	}
	conn, _, err := websocket.DefaultDialer.Dial(c.conf.API, header) // nolint
	if err != nil {
		log.Warnf("连接到反向WebSocket API服务器 %v 时出现错误: %v", c.conf.API, err)
		if c.conf.ReconnectInterval != 0 {
			time.Sleep(time.Millisecond * time.Duration(c.conf.ReconnectInterval))
			c.connectAPI()
		}
		return
	}
	log.Infof("已连接到反向WebSocket API服务器 %v", c.conf.API)
	wrappedConn := &webSocketConn{Conn: conn, apiCaller: newAPICaller(c.bot)}
	if c.conf.RateLimit.Enabled {
		wrappedConn.apiCaller.use(rateLimit(c.conf.RateLimit.Frequency, c.conf.RateLimit.Bucket))
	}
	go c.listenAPI(wrappedConn, false)
}

func (c *websocketClient) connectEvent() {
	log.Infof("开始尝试连接到反向WebSocket Event服务器: %v", c.conf.Event)
	header := http.Header{
		"X-Client-Role": []string{"Event"},
		"X-Self-ID":     []string{strconv.FormatInt(c.bot.Client.Uin, 10)},
		"User-Agent":    []string{"CQHttp/4.15.0"},
	}
	if c.token != "" {
		header["Authorization"] = []string{"Token " + c.token}
	}
	conn, _, err := websocket.DefaultDialer.Dial(c.conf.Event, header) // nolint
	if err != nil {
		log.Warnf("连接到反向WebSocket Event服务器 %v 时出现错误: %v", c.conf.Event, err)
		if c.conf.ReconnectInterval != 0 {
			time.Sleep(time.Millisecond * time.Duration(c.conf.ReconnectInterval))
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

	log.Infof("已连接到反向WebSocket Event服务器 %v", c.conf.Event)
	if c.eventConn == nil {
		wrappedConn := &webSocketConn{Conn: conn, apiCaller: newAPICaller(c.bot)}
		c.eventConn = wrappedConn
	} else {
		c.eventConn.Conn = conn
	}
}

func (c *websocketClient) connectUniversal() {
	log.Infof("开始尝试连接到反向WebSocket Universal服务器: %v", c.conf.Universal)
	header := http.Header{
		"X-Client-Role": []string{"Universal"},
		"X-Self-ID":     []string{strconv.FormatInt(c.bot.Client.Uin, 10)},
		"User-Agent":    []string{"CQHttp/4.15.0"},
	}
	if c.token != "" {
		header["Authorization"] = []string{"Token " + c.token}
	}
	conn, _, err := websocket.DefaultDialer.Dial(c.conf.Universal, header) // nolint
	if err != nil {
		log.Warnf("连接到反向WebSocket Universal服务器 %v 时出现错误: %v", c.conf.Universal, err)
		if c.conf.ReconnectInterval != 0 {
			time.Sleep(time.Millisecond * time.Duration(c.conf.ReconnectInterval))
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

	if c.universalConn == nil {
		wrappedConn := &webSocketConn{Conn: conn, apiCaller: newAPICaller(c.bot)}
		if c.conf.RateLimit.Enabled {
			wrappedConn.apiCaller.use(rateLimit(c.conf.RateLimit.Frequency, c.conf.RateLimit.Bucket))
		}
		c.universalConn = wrappedConn
	} else {
		c.universalConn.Conn = conn
	}
	go c.listenAPI(c.universalConn, true)
}

func (c *websocketClient) listenAPI(conn *webSocketConn, u bool) {
	defer func() { _ = conn.Close() }()
	for {
		buffer := global.NewBuffer()
		t, reader, err := conn.NextReader()
		if err != nil {
			log.Warnf("监听反向WS API时出现错误: %v", err)
			break
		}
		_, err = buffer.ReadFrom(reader)
		if err != nil {
			log.Warnf("监听反向WS API时出现错误: %v", err)
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
	if c.conf.ReconnectInterval != 0 {
		time.Sleep(time.Millisecond * time.Duration(c.conf.ReconnectInterval))
		if !u {
			go c.connectAPI()
		}
	}
}

func (c *websocketClient) onBotPushEvent(e *coolq.Event) {
	filter := findFilter(c.filter)
	if filter != nil && !filter.Eval(gjson.Parse(e.JSONString())) {
		log.Debugf("上报Event %s 到 WS服务器 时被过滤.", e.JSONBytes())
		return
	}
	push := func(conn *webSocketConn, reconnect func()) {
		log.Debugf("向WS服务器 %v 推送Event: %s", conn.RemoteAddr().String(), e.JSONBytes())
		conn.Lock()
		defer conn.Unlock()
		_ = conn.SetWriteDeadline(time.Now().Add(time.Second * 15))
		if err := conn.WriteMessage(websocket.TextMessage, e.JSONBytes()); err != nil {
			log.Warnf("向WS服务器 %v 推送Event时出现错误: %v", conn.RemoteAddr().String(), err)
			_ = conn.Close()
			if c.conf.ReconnectInterval != 0 {
				time.Sleep(time.Millisecond * time.Duration(c.conf.ReconnectInterval))
				reconnect()
			}
		}
	}
	if c.eventConn != nil {
		push(c.eventConn, c.connectEvent)
	}
	if c.universalConn != nil {
		push(c.universalConn, c.connectUniversal)
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

	conn := &webSocketConn{Conn: c, apiCaller: newAPICaller(s.bot)}

	s.eventConnMutex.Lock()
	s.eventConn = append(s.eventConn, conn)
	s.eventConnMutex.Unlock()
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
	conn := &webSocketConn{Conn: c, apiCaller: newAPICaller(s.bot)}
	if s.conf.RateLimit.Enabled {
		conn.apiCaller.use(rateLimit(s.conf.RateLimit.Frequency, s.conf.RateLimit.Bucket))
	}
	go s.listenAPI(conn)
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
	conn := &webSocketConn{Conn: c, apiCaller: newAPICaller(s.bot)}
	if s.conf.RateLimit.Enabled {
		conn.apiCaller.use(rateLimit(s.conf.RateLimit.Frequency, s.conf.RateLimit.Bucket))
	}
	s.eventConnMutex.Lock()
	s.eventConn = append(s.eventConn, conn)
	s.eventConnMutex.Unlock()
	s.listenAPI(conn)
}

func (s *webSocketServer) listenAPI(c *webSocketConn) {
	defer func() { _ = c.Close() }()
	for {
		buffer := global.NewBuffer()
		t, reader, err := c.NextReader()
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

func (c *webSocketConn) handleRequest(_ *coolq.CQBot, payload []byte) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("处置WS命令时发生无法恢复的异常：%v\n%s", err, debug.Stack())
			_ = c.Close()
		}
	}()
	j := gjson.Parse(utils.B2S(payload))
	t := strings.TrimSuffix(j.Get("action").Str, "_async")
	log.Debugf("WS接收到API调用: %v 参数: %v", t, j.Get("params").Raw)
	ret := c.apiCaller.callAPI(t, j.Get("params"))
	if j.Get("echo").Exists() {
		ret["echo"] = j.Get("echo").Value()
	}
	c.Lock()
	defer c.Unlock()
	_ = c.WriteJSON(ret)
}

func (s *webSocketServer) onBotPushEvent(e *coolq.Event) {
	s.eventConnMutex.Lock()
	defer s.eventConnMutex.Unlock()

	filter := findFilter(s.filter)
	if filter != nil && !filter.Eval(gjson.Parse(e.JSONString())) {
		log.Debugf("上报Event %s 到 WS客户端 时被过滤.", e.JSONBytes())
		return
	}
	j := 0
	for i := 0; i < len(s.eventConn); i++ {
		conn := s.eventConn[i]
		log.Debugf("向WS客户端 %v 推送Event: %s", conn.RemoteAddr().String(), e.JSONBytes())
		conn.Lock()
		if err := conn.WriteMessage(websocket.TextMessage, e.JSONBytes()); err != nil {
			_ = conn.Close()
			conn = nil
			continue
		}
		if i != j {
			// i != j means that some connection has been closed.
			// use a in-place removal to avoid copying.
			s.eventConn[j] = conn
		}
		conn.Unlock()
		j++
	}
	s.eventConn = s.eventConn[:j]
}
