package server

import (
	"fmt"
	"github.com/Mrs4s/go-cqhttp/coolq"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"net/http"
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
			log.Infof("API调用: %v", j.Get("action").Str)
			if f, ok := wsApi[t]; ok {
				ret := f(s.bot, j.Get("params"))
				if j.Get("echo").Exists() {
					ret["echo"] = j.Get("echo").Value()
				}
				_ = c.WriteJSON(ret)
			}
		}
	}
}

func (s *websocketServer) onBotPushEvent(m coolq.MSG) {
	s.pushLock.Lock()
	defer s.pushLock.Unlock()
	pos := 0
	for _, conn := range s.eventConn {
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
		if p.Get("group_id").Int() != 0 {
			return bot.CQSendGroupMessage(p.Get("group_id").Int(), p.Get("message").Str)
		}
		if p.Get("user_id").Int() != 0 {
			return bot.CQSendPrivateMessage(p.Get("user_id").Int(), p.Get("message").Str)
		}
		return coolq.MSG{}
	},
	"send_group_msg": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQSendGroupMessage(p.Get("group_id").Int(), p.Get("message").Str)
	},
	"send_private_msg": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQSendPrivateMessage(p.Get("user_id").Int(), p.Get("message").Str)
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
		return bot.CQSetGroupBan(p.Get("group_id").Int(), p.Get("user_id").Int(), uint32(p.Get("duration").Int()))
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
	"get_image": func(bot *coolq.CQBot, p gjson.Result) coolq.MSG {
		return bot.CQGetImage(p.Get("file").Str)
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
