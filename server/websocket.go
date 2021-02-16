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

//WebSocketClient Websocket客户端实例
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
}

//WebSocketServer 初始化一个WebSocketServer实例
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
		log.Infof("CQ Websocket 服务器已启动: %v", addr)
		log.Fatal(http.ListenAndServe(addr, nil))
	}()
}

//NewWebSocketClient 初始化一个NWebSocket客户端
func NewWebSocketClient(conf *global.GoCQReverseWebSocketConfig, authToken string, b *coolq.CQBot) *WebSocketClient {
	return &WebSocketClient{conf: conf, token: authToken, bot: b}
}

//Run 运行实例
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
	log.Infof("开始尝试连接到反向Websocket API服务器: %v", c.conf.ReverseAPIURL)
	header := http.Header{
		"X-Client-Role": []string{"API"},
		"X-Self-ID":     []string{strconv.FormatInt(c.bot.Client.Uin, 10)},
		"User-Agent":    []string{"CQHttp/4.15.0"},
	}
	if c.token != "" {
		header["Authorization"] = []string{"Token " + c.token}
	}
	conn, _, err := websocket.DefaultDialer.Dial(c.conf.ReverseAPIURL, header)
	if err != nil {
		log.Warnf("连接到反向Websocket API服务器 %v 时出现错误: %v", c.conf.ReverseAPIURL, err)
		if c.conf.ReverseReconnectInterval != 0 {
			time.Sleep(time.Millisecond * time.Duration(c.conf.ReverseReconnectInterval))
			c.connectAPI()
		}
		return
	}
	log.Infof("已连接到反向Websocket API服务器 %v", c.conf.ReverseAPIURL)
	wrappedConn := &webSocketConn{Conn: conn}
	go c.listenAPI(wrappedConn, false)
}

func (c *WebSocketClient) connectEvent() {
	log.Infof("开始尝试连接到反向Websocket Event服务器: %v", c.conf.ReverseEventURL)
	header := http.Header{
		"X-Client-Role": []string{"Event"},
		"X-Self-ID":     []string{strconv.FormatInt(c.bot.Client.Uin, 10)},
		"User-Agent":    []string{"CQHttp/4.15.0"},
	}
	if c.token != "" {
		header["Authorization"] = []string{"Token " + c.token}
	}
	conn, _, err := websocket.DefaultDialer.Dial(c.conf.ReverseEventURL, header)
	if err != nil {
		log.Warnf("连接到反向Websocket Event服务器 %v 时出现错误: %v", c.conf.ReverseEventURL, err)
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
		log.Warnf("反向Websocket 握手时出现错误: %v", err)
	}

	log.Infof("已连接到反向Websocket Event服务器 %v", c.conf.ReverseEventURL)
	c.eventConn = &webSocketConn{Conn: conn}
}

func (c *WebSocketClient) connectUniversal() {
	log.Infof("开始尝试连接到反向Websocket Universal服务器: %v", c.conf.ReverseURL)
	header := http.Header{
		"X-Client-Role": []string{"Universal"},
		"X-Self-ID":     []string{strconv.FormatInt(c.bot.Client.Uin, 10)},
		"User-Agent":    []string{"CQHttp/4.15.0"},
	}
	if c.token != "" {
		header["Authorization"] = []string{"Token " + c.token}
	}
	conn, _, err := websocket.DefaultDialer.Dial(c.conf.ReverseURL, header)
	if err != nil {
		log.Warnf("连接到反向Websocket Universal服务器 %v 时出现错误: %v", c.conf.ReverseURL, err)
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
		log.Warnf("反向Websocket 握手时出现错误: %v", err)
	}

	wrappedConn := &webSocketConn{Conn: conn}
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
		log.Debugf("向WS服务器 %v 推送Event: %v", c.eventConn.RemoteAddr().String(), m.ToJson())
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
		log.Debugf("向WS服务器 %v 推送Event: %v", c.universalConn.RemoteAddr().String(), m.ToJson())
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
				log.Warnf("已拒绝 %v 的 Websocket 请求: Token鉴权失败", r.RemoteAddr)
				w.WriteHeader(401)
				return
			}
		}
	}
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Warnf("处理 Websocket 请求时出现错误: %v", err)
		return
	}
	err = c.WriteMessage(websocket.TextMessage, []byte(s.handshake))
	if err != nil {
		log.Warnf("Websocket 握手时出现错误: %v", err)
		c.Close()
		return
	}

	log.Infof("接受 Websocket 连接: %v (/event)", r.RemoteAddr)

	conn := &webSocketConn{Conn: c}

	s.eventConnMutex.Lock()
	s.eventConn = append(s.eventConn, conn)
	s.eventConnMutex.Unlock()
}

func (s *webSocketServer) api(w http.ResponseWriter, r *http.Request) {
	if s.token != "" {
		if auth := r.URL.Query().Get("access_token"); auth != s.token {
			if auth := strings.SplitN(r.Header.Get("Authorization"), " ", 2); len(auth) != 2 || auth[1] != s.token {
				log.Warnf("已拒绝 %v 的 Websocket 请求: Token鉴权失败", r.RemoteAddr)
				w.WriteHeader(401)
				return
			}
		}
	}
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Warnf("处理 Websocket 请求时出现错误: %v", err)
		return
	}
	log.Infof("接受 Websocket 连接: %v (/api)", r.RemoteAddr)
	conn := &webSocketConn{Conn: c}
	go s.listenAPI(conn)
}

func (s *webSocketServer) any(w http.ResponseWriter, r *http.Request) {
	if s.token != "" {
		if auth := r.URL.Query().Get("access_token"); auth != s.token {
			if auth := strings.SplitN(r.Header.Get("Authorization"), " ", 2); len(auth) != 2 || auth[1] != s.token {
				log.Warnf("已拒绝 %v 的 Websocket 请求: Token鉴权失败", r.RemoteAddr)
				w.WriteHeader(401)
				return
			}
		}
	}
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Warnf("处理 Websocket 请求时出现错误: %v", err)
		return
	}
	err = c.WriteMessage(websocket.TextMessage, []byte(s.handshake))
	if err != nil {
		log.Warnf("Websocket 握手时出现错误: %v", err)
		c.Close()
		return
	}
	log.Infof("接受 Websocket 连接: %v (/)", r.RemoteAddr)
	conn := &webSocketConn{Conn: c}
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

func (c *webSocketConn) handleRequest(bot *coolq.CQBot, payload []byte) {
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
	if f, ok := wsAPI[t]; ok {
		ret := f(bot, j.Get("params"))
		if j.Get("echo").Exists() {
			ret["echo"] = j.Get("echo").Value()
		}
		c.Lock()
		defer c.Unlock()
		_ = c.WriteJSON(ret)
	} else {
		ret := coolq.Failed(1404, "API_NOT_FOUND", "API不存在")
		if j.Get("echo").Exists() {
			ret["echo"] = j.Get("echo").Value()
		}
		c.Lock()
		defer c.Unlock()
		_ = c.WriteJSON(ret)
	}
}

func (s *webSocketServer) onBotPushEvent(m coolq.MSG) {
	s.eventConnMutex.Lock()
	defer s.eventConnMutex.Unlock()
	for i, l := 0, len(s.eventConn); i < l; i++ {
		conn := s.eventConn[i]
		log.Debugf("向WS客户端 %v 推送Event: %v", conn.RemoteAddr().String(), m.ToJson())
		conn.Lock()
		if err := conn.WriteMessage(websocket.TextMessage, []byte(m.ToJson())); err != nil {
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

var wsAPI = map[string]func(*coolq.CQBot, gjson.Result) coolq.MSG{
	"get_login_info": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQGetLoginInfo()
	},
	"get_friend_list": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQGetFriendList()
	},
	"get_group_list": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQGetGroupList(p.Get("no_cache").Bool())
	},
	"get_group_info": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQGetGroupInfo(p.Get("group_id").Int(), p.Get("no_cache").Bool())
	},
	"get_group_member_list": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQGetGroupMemberList(p.Get("group_id").Int(), p.Get("no_cache").Bool())
	},
	"get_group_member_info": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQGetGroupMemberInfo(
			p.Get("group_id").Int(), p.Get("user_id").Int(),
		)
	},
	"send_msg": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		autoEscape := global.EnsureBool(p.Get("auto_escape"), false)
		if p.Get("message_type").Str == "private" {
			return bot.CQSendPrivateMessage(p.Get("user_id").Int(), p.Get("message"), autoEscape)
		}
		if p.Get("message_type").Str == "group" {
			return bot.CQSendGroupMessage(p.Get("group_id").Int(), p.Get("message"), autoEscape)
		}
		if p.Get("group_id").Int() != 0 {
			return bot.CQSendGroupMessage(p.Get("group_id").Int(), p.Get("message"), autoEscape)
		}
		if p.Get("user_id").Int() != 0 {
			return bot.CQSendPrivateMessage(p.Get("user_id").Int(), p.Get("message"), autoEscape)
		}
		return coolq.MSG{}
	},
	"send_group_msg": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQSendGroupMessage(p.Get("group_id").Int(), p.Get("message"), global.EnsureBool(p.Get("auto_escape"), false))
	},
	"send_group_forward_msg": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQSendGroupForwardMessage(p.Get("group_id").Int(), p.Get("messages"))
	},
	"send_private_msg": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQSendPrivateMessage(p.Get("user_id").Int(), p.Get("message"), global.EnsureBool(p.Get("auto_escape"), false))
	},
	"delete_msg": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQDeleteMessage(int32(p.Get("message_id").Int()))
	},
	"set_friend_add_request": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		apr := true
		if p.Get("approve").Exists() {
			apr = p.Get("approve").Bool()
		}
		return bot.CQProcessFriendRequest(p.Get("flag").Str, apr)
	},
	"set_group_add_request": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		subType := p.Get("sub_type").Str
		apr := true
		if subType == "" {
			subType = p.Get("type").Str
		}
		if p.Get("approve").Exists() {
			apr = p.Get("approve").Bool()
		}
		return bot.CQProcessGroupRequest(p.Get("flag").Str, subType, p.Get("reason").Str, apr)
	},
	"set_group_card": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQSetGroupCard(p.Get("group_id").Int(), p.Get("user_id").Int(), p.Get("card").Str)
	},
	"set_group_special_title": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQSetGroupSpecialTitle(p.Get("group_id").Int(), p.Get("user_id").Int(), p.Get("special_title").Str)
	},
	"set_group_kick": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQSetGroupKick(p.Get("group_id").Int(), p.Get("user_id").Int(), p.Get("message").Str, p.Get("reject_add_request").Bool())
	},
	"set_group_ban": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQSetGroupBan(p.Get("group_id").Int(), p.Get("user_id").Int(), func() uint32 {
			if p.Get("duration").Exists() {
				return uint32(p.Get("duration").Int())
			}
			return 1800
		}())
	},
	"set_group_whole_ban": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQSetGroupWholeBan(p.Get("group_id").Int(), func() bool {
			if p.Get("enable").Exists() {
				return p.Get("enable").Bool()
			}
			return true
		}())
	},
	"set_group_name": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQSetGroupName(p.Get("group_id").Int(), p.Get("group_name").Str)
	},
	"set_group_admin": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQSetGroupAdmin(p.Get("group_id").Int(), p.Get("user_id").Int(), func() bool {
			if p.Get("enable").Exists() {
				return p.Get("enable").Bool()
			}
			return true
		}())
	},
	"_send_group_notice": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQSetGroupMemo(p.Get("group_id").Int(), p.Get("content").Str)
	},
	"set_group_leave": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQSetGroupLeave(p.Get("group_id").Int())
	},
	"get_image": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQGetImage(p.Get("file").Str)
	},
	"get_forward_msg": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		id := p.Get("message_id").Str
		if id == "" {
			id = p.Get("id").Str
		}
		return bot.CQGetForwardMessage(id)
	},
	"get_msg": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQGetMessage(int32(p.Get("message_id").Int()))
	},
	"download_file": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		headers := map[string]string{}
		headersToken := p.Get("headers")
		if headersToken.IsArray() {
			for _, sub := range headersToken.Array() {
				str := strings.SplitN(sub.String(), "=", 2)
				if len(str) == 2 {
					headers[str[0]] = str[1]
				}
			}
		}
		if headersToken.Type == gjson.String {
			lines := strings.Split(headersToken.String(), "\r\n")
			for _, sub := range lines {
				str := strings.SplitN(sub, "=", 2)
				if len(str) == 2 {
					headers[str[0]] = str[1]
				}
			}
		}
		return bot.CQDownloadFile(p.Get("url").Str, headers, int(p.Get("thread_count").Int()))
	},
	"get_group_honor_info": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQGetGroupHonorInfo(p.Get("group_id").Int(), p.Get("type").Str)
	},
	"set_restart": func(c *coolq.CQBot, p gjson.Result) coolq.MSG {
		var delay int64
		delay = p.Get("delay").Int()
		if delay < 0 {
			delay = 0
		}
		defer func(delay int64) {
			time.Sleep(time.Duration(delay) * time.Millisecond)
			Restart <- struct{}{}
		}(delay)
		return coolq.MSG{"data": nil, "retcode": 0, "status": "async"}

	},
	"can_send_image": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQCanSendImage()
	},
	"can_send_record": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQCanSendRecord()
	},
	"get_stranger_info": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQGetStrangerInfo(p.Get("user_id").Int())
	},
	"get_status": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQGetStatus()
	},
	"get_version_info": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQGetVersionInfo()
	},
	"get_group_system_msg": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQGetGroupSystemMessages()
	},
	"get_group_file_system_info": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQGetGroupFileSystemInfo(p.Get("group_id").Int())
	},
	"get_group_root_files": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQGetGroupRootFiles(p.Get("group_id").Int())
	},
	"get_group_files_by_folder": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQGetGroupFilesByFolderId(p.Get("group_id").Int(), p.Get("folder_id").Str)
	},
	"get_group_file_url": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQGetGroupFileUrl(p.Get("group_id").Int(), p.Get("file_id").Str, int32(p.Get("busid").Int()))
	},
	"upload_group_file": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQUploadGroupFile(p.Get("group_id").Int(), p.Get("file").Str, p.Get("name").Str, p.Get("folder").Str)
	},
	"get_group_msg_history": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQGetGroupMessageHistory(p.Get("group_id").Int(), p.Get("message_seq").Int())
	},
	"_get_vip_info": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQGetVipInfo(p.Get("user_id").Int())
	},
	"reload_event_filter": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQReloadEventFilter()
	},
	".ocr_image": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQOcrImage(p.Get("image").Str)
	},
	"ocr_image": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQOcrImage(p.Get("image").Str)
	},
	"get_group_at_all_remain": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQGetAtAllRemain(p.Get("group_id").Int())
	},
	"get_online_clients": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQGetOnlineClients(p.Get("no_cache").Bool())
	},
	".get_word_slices": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQGetWordSlices(p.Get("content").Str)
	},
	"set_group_portrait": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQSetGroupPortrait(p.Get("group_id").Int(), p.Get("file").String(), p.Get("cache").String())
	},
	"set_essence_msg": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQSetEssenceMessage(int32(p.Get("message_id").Int()))
	},
	"delete_essence_msg": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQDeleteEssenceMessage(int32(p.Get("message_id").Int()))
	},
	"get_essence_msg_list": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQGetEssenceMessageList(p.Get("group_id").Int())
	},
	"check_url_safely": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQCheckUrlSafely(p.Get("url").String())
	},
	"set_group_anonymous_ban": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		obj := p.Get("anonymous")
		flag := p.Get("anonymous_flag")
		if !flag.Exists() {
			flag = p.Get("flag")
		}
		if !flag.Exists() && !obj.Exists() {
			return coolq.Failed(100, "FLAG_NOT_FOUND", "flag未找到")
		}
		if !flag.Exists() {
			flag = obj.Get("flag")
		}
		return bot.CQSetGroupAnonymousBan(p.Get("group_id").Int(), flag.String(), int32(p.Get("duration").Int()))
	},
	".handle_quick_operation": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQHandleQuickOperation(p.Get("context"), p.Get("operation"))
	},
}
