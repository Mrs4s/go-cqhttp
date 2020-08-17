package server

import (
	"fmt"
	"github.com/Mrs4s/go-cqhttp/coolq"
	"github.com/Mrs4s/go-cqhttp/global"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	wsc "golang.org/x/net/websocket"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type websocketServer struct {
	bot       *coolq.CQBot
	token     string
	eventConn []*websocket.Conn
	pushLock  *sync.Mutex
	handshake string
}

type websocketClient struct {
	conf  *global.GoCQReverseWebsocketConfig
	token string
	bot   *coolq.CQBot

	pushLock      *sync.Mutex
	universalConn *wsc.Conn
	eventConn     *wsc.Conn
}

var WebsocketServer = &websocketServer{}
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (s *websocketServer) Run(addr, authToken string, b *coolq.CQBot) {
	s.token = authToken
	s.pushLock = new(sync.Mutex)
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

func NewWebsocketClient(conf *global.GoCQReverseWebsocketConfig, authToken string, b *coolq.CQBot) *websocketClient {
	return &websocketClient{conf: conf, token: authToken, bot: b, pushLock: new(sync.Mutex)}
}

func (c *websocketClient) Run() {
	if !c.conf.Enabled {
		return
	}
	if c.conf.ReverseUrl != "" {
		c.connectUniversal()
	} else {
		if c.conf.ReverseApiUrl != "" {
			c.connectApi()
		}
		if c.conf.ReverseEventUrl != "" {
			c.connectEvent()
		}
	}
	c.bot.OnEventPush(c.onBotPushEvent)
}

func (c *websocketClient) connectApi() {
	log.Infof("开始尝试连接到反向Websocket API服务器: %v", c.conf.ReverseApiUrl)
	wsConf, err := wsc.NewConfig(c.conf.ReverseApiUrl, c.conf.ReverseApiUrl)
	if err != nil {
		log.Warnf("连接到反向Websocket API服务器 %v 时出现致命错误: %v", c.conf.ReverseApiUrl, err)
		return
	}
	wsConf.Header["X-Client-Role"] = []string{"API"}
	wsConf.Header["X-Self-ID"] = []string{strconv.FormatInt(c.bot.Client.Uin, 10)}
	wsConf.Header["User-Agent"] = []string{"CQHttp/4.15.0"}
	if c.token != "" {
		wsConf.Header["Authorization"] = []string{"Token " + c.token}
	}
	conn, err := wsc.DialConfig(wsConf)
	if err != nil {
		log.Warnf("连接到反向Websocket API服务器 %v 时出现错误: %v", c.conf.ReverseApiUrl, err)
		if c.conf.ReverseReconnectInterval != 0 {
			time.Sleep(time.Millisecond * time.Duration(c.conf.ReverseReconnectInterval))
			c.connectApi()
		}
		return
	}
	log.Infof("已连接到反向Websocket API服务器 %v", c.conf.ReverseApiUrl)
	go c.listenApi(conn, false)
}

func (c *websocketClient) connectEvent() {
	log.Infof("开始尝试连接到反向Websocket Event服务器: %v", c.conf.ReverseEventUrl)
	wsConf, err := wsc.NewConfig(c.conf.ReverseEventUrl, c.conf.ReverseEventUrl)
	if err != nil {
		log.Warnf("连接到反向Websocket Event服务器 %v 时出现致命错误: %v", c.conf.ReverseApiUrl, err)
		return
	}
	wsConf.Header["X-Client-Role"] = []string{"Event"}
	wsConf.Header["X-Self-ID"] = []string{strconv.FormatInt(c.bot.Client.Uin, 10)}
	wsConf.Header["User-Agent"] = []string{"CQHttp/4.15.0"}
	if c.token != "" {
		wsConf.Header["Authorization"] = []string{"Token " + c.token}
	}
	conn, err := wsc.DialConfig(wsConf)
	if err != nil {
		log.Warnf("连接到反向Websocket API服务器 %v 时出现错误: %v", c.conf.ReverseApiUrl, err)
		if c.conf.ReverseReconnectInterval != 0 {
			time.Sleep(time.Millisecond * time.Duration(c.conf.ReverseReconnectInterval))
			c.connectApi()
		}
		return
	}
	log.Infof("已连接到反向Websocket Event服务器 %v", c.conf.ReverseEventUrl)
	c.eventConn = conn
}

func (c *websocketClient) connectUniversal() {
	log.Infof("开始尝试连接到反向Websocket Universal服务器: %v", c.conf.ReverseUrl)
	wsConf, err := wsc.NewConfig(c.conf.ReverseUrl, c.conf.ReverseUrl)
	if err != nil {
		log.Warnf("连接到反向Websocket Universal服务器 %v 时出现致命错误: %v", c.conf.ReverseUrl, err)
		return
	}
	wsConf.Header["X-Client-Role"] = []string{"Universal"}
	wsConf.Header["X-Self-ID"] = []string{strconv.FormatInt(c.bot.Client.Uin, 10)}
	wsConf.Header["User-Agent"] = []string{"CQHttp/4.15.0"}
	if c.token != "" {
		wsConf.Header["Authorization"] = []string{"Token " + c.token}
	}
	conn, err := wsc.DialConfig(wsConf)
	if err != nil {
		log.Warnf("连接到反向Websocket Universal服务器 %v 时出现错误: %v", c.conf.ReverseUrl, err)
		if c.conf.ReverseReconnectInterval != 0 {
			time.Sleep(time.Millisecond * time.Duration(c.conf.ReverseReconnectInterval))
			c.connectUniversal()
		}
		return
	}
	go c.listenApi(conn, true)
	c.universalConn = conn
}

func (c *websocketClient) listenApi(conn *wsc.Conn, u bool) {
	defer conn.Close()
	for {
		var buf []byte
		err := wsc.Message.Receive(conn, &buf)
		if err != nil {
			break
		}
		j := gjson.ParseBytes(buf)
		t := strings.ReplaceAll(j.Get("action").Str, "_async", "")
		log.Debugf("反向WS接收到API调用: %v 参数: %v", t, j.Get("params").Raw)
		if f, ok := wsApi[t]; ok {
			ret := f(c.bot, j.Get("params"))
			if j.Get("echo").Exists() {
				ret["echo"] = j.Get("echo").Value()
			}
			c.pushLock.Lock()
			log.Debugf("准备发送API %v 处理结果: %v", t, ret.ToJson())
			_, _ = conn.Write([]byte(ret.ToJson()))
			c.pushLock.Unlock()
		}
	}
	if c.conf.ReverseReconnectInterval != 0 {
		time.Sleep(time.Millisecond * time.Duration(c.conf.ReverseReconnectInterval))
		if !u {
			c.connectApi()
		}
	}
}

func (c *websocketClient) onBotPushEvent(m coolq.MSG) {
	c.pushLock.Lock()
	defer c.pushLock.Unlock()
	if c.eventConn != nil {
		log.Debugf("向WS服务器 %v 推送Event: %v", c.eventConn.RemoteAddr().String(), m.ToJson())
		if _, err := c.eventConn.Write([]byte(m.ToJson())); err != nil {
			_ = c.eventConn.Close()
			if c.conf.ReverseReconnectInterval != 0 {
				go func() {
					time.Sleep(time.Millisecond * time.Duration(c.conf.ReverseReconnectInterval))
					c.connectEvent()
				}()
			}
		}
	}
	if c.universalConn != nil {
		log.Debugf("向WS服务器 %v 推送Event: %v", c.universalConn.RemoteAddr().String(), m.ToJson())
		if _, err := c.universalConn.Write([]byte(m.ToJson())); err != nil {
			_ = c.universalConn.Close()
			if c.conf.ReverseReconnectInterval != 0 {
				go func() {
					time.Sleep(time.Millisecond * time.Duration(c.conf.ReverseReconnectInterval))
					c.connectUniversal()
				}()
			}
		}
	}
}

func (s *websocketServer) event(w http.ResponseWriter, r *http.Request) {
	if s.token != "" {
		if r.URL.Query().Get("access_token") != s.token && strings.SplitN(r.Header.Get("Authorization"), " ", 2)[1] != s.token {
			log.Warnf("已拒绝 %v 的 Websocket 请求: Token错误", r.RemoteAddr)
			w.WriteHeader(401)
			return
		}
	}
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Warnf("处理 Websocket 请求时出现错误: %v", err)
		return
	}
	err = c.WriteMessage(websocket.TextMessage, []byte(s.handshake))
	if err == nil {
		log.Infof("接受 Websocket 连接: %v (/event)", r.RemoteAddr)
		s.eventConn = append(s.eventConn, c)
	}
}

func (s *websocketServer) api(w http.ResponseWriter, r *http.Request) {
	if s.token != "" {
		if r.URL.Query().Get("access_token") != s.token && strings.SplitN(r.Header.Get("Authorization"), " ", 2)[1] != s.token {
			log.Warnf("已拒绝 %v 的 Websocket 请求: Token错误", r.RemoteAddr)
			w.WriteHeader(401)
			return
		}
	}
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Warnf("处理 Websocket 请求时出现错误: %v", err)
		return
	}
	log.Infof("接受 Websocket 连接: %v (/api)", r.RemoteAddr)
	go s.listenApi(c)
}

func (s *websocketServer) any(w http.ResponseWriter, r *http.Request) {
	if s.token != "" {
		if r.URL.Query().Get("access_token") != s.token && strings.SplitN(r.Header.Get("Authorization"), " ", 2)[1] != s.token {
			log.Warnf("已拒绝 %v 的 Websocket 请求: Token错误", r.RemoteAddr)
			w.WriteHeader(401)
			return
		}
	}
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Warnf("处理 Websocket 请求时出现错误: %v", err)
		return
	}
	err = c.WriteMessage(websocket.TextMessage, []byte(s.handshake))
	if err == nil {
		log.Infof("接受 Websocket 连接: %v (/)", r.RemoteAddr)
		s.eventConn = append(s.eventConn, c)
		s.listenApi(c)
	}
}

func (s *websocketServer) listenApi(c *websocket.Conn) {
	defer c.Close()
	for {
		t, payload, err := c.ReadMessage()
		if err != nil {
			break
		}
		if t == websocket.TextMessage {
			j := gjson.ParseBytes(payload)
			t := strings.ReplaceAll(j.Get("action").Str, "_async", "") //TODO: async support
			log.Debugf("WS接收到API调用: %v 参数: %v", t, j.Get("params").Raw)
			if f, ok := wsApi[t]; ok {
				ret := f(s.bot, j.Get("params"))
				if j.Get("echo").Exists() {
					ret["echo"] = j.Get("echo").Value()
				}
				s.pushLock.Lock()
				_ = c.WriteJSON(ret)
				s.pushLock.Unlock()
			}
		}
	}
}

func (s *websocketServer) onBotPushEvent(m coolq.MSG) {
	s.pushLock.Lock()
	defer s.pushLock.Unlock()
	pos := 0
	for _, conn := range s.eventConn {
		log.Debugf("向WS客户端 %v 推送Event: %v", conn.RemoteAddr().String(), m.ToJson())
		err := conn.WriteMessage(websocket.TextMessage, []byte(m.ToJson()))
		if err != nil {
			_ = conn.Close()
			s.eventConn = append(s.eventConn[:pos], s.eventConn[pos+1:]...)
			if pos > 0 {
				pos++
			}
		}
		pos++
	}
}

var wsApi = map[string]func(*coolq.CQBot, gjson.Result) coolq.MSG{
	"get_login_info": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQGetLoginInfo()
	},
	"get_friend_list": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQGetFriendList()
	},
	"get_group_list": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQGetGroupList()
	},
	"get_group_info": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQGetGroupInfo(p.Get("group_id").Int())
	},
	"get_group_member_list": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQGetGroupMemberList(p.Get("group_id").Int())
	},
	"get_group_member_info": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQGetGroupMemberInfo(
			p.Get("group_id").Int(), p.Get("user_id").Int(),
			p.Get("no_cache").Bool(),
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
		return bot.CQProcessGroupRequest(p.Get("flag").Str, subType, apr)
	},
	"set_group_card": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQSetGroupCard(p.Get("group_id").Int(), p.Get("user_id").Int(), p.Get("card").Str)
	},
	"set_group_special_title": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQSetGroupSpecialTitle(p.Get("group_id").Int(), p.Get("user_id").Int(), p.Get("special_title").Str)
	},
	"set_group_kick": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQSetGroupKick(p.Get("group_id").Int(), p.Get("user_id").Int(), p.Get("message").Str)
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
		return bot.CQSetGroupName(p.Get("group_id").Int(), p.Get("name").Str)
	},
	"set_group_leave": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQSetGroupLeave(p.Get("group_id").Int())
	},
	"get_image": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQGetImage(p.Get("file").Str)
	},
	"get_forward_msg": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQGetForwardMessage(p.Get("message_id").Str)
	},
	"get_group_msg": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQGetGroupMessage(int32(p.Get("message_id").Int()))
	},
	"can_send_image": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQCanSendImage()
	},
	"can_send_record": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQCanSendRecord()
	},
	"get_status": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQGetStatus()
	},
	"get_version_info": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQGetVersionInfo()
	},
	".handle_quick_operation": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQHandleQuickOperation(p.Get("context"), p.Get("operation"))
	},
}
