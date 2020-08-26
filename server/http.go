package server

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Mrs4s/go-cqhttp/coolq"
	"github.com/Mrs4s/go-cqhttp/global"
	"github.com/gin-gonic/gin"
	"github.com/guonaihong/gout"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

type httpServer struct {
	engine *gin.Engine
	bot    *coolq.CQBot
}

type httpClient struct {
	bot     *coolq.CQBot
	secret  string
	addr    string
	timeout int32
}

var HttpServer = &httpServer{}

func (s *httpServer) Run(addr, authToken string, bot *coolq.CQBot) {
	gin.SetMode(gin.ReleaseMode)
	s.engine = gin.New()
	s.bot = bot
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
			if auth := c.Request.Header.Get("Authorization"); auth != "" {
				if strings.SplitN(auth, " ", 2)[1] != authToken {
					c.AbortWithStatus(401)
					return
				}
			} else if c.Query("access_token") != authToken {
				c.AbortWithStatus(401)
				return
			} else {
				c.Next()
			}
		})
	}

	s.engine.Any("/get_login_info", s.GetLoginInfo)
	s.engine.Any("/get_login_info_async", s.GetLoginInfo)

	s.engine.Any("/get_friend_list", s.GetFriendList)
	s.engine.Any("/get_friend_list_async", s.GetFriendList)

	s.engine.Any("/get_group_list", s.GetGroupList)
	s.engine.Any("/get_group_list_async", s.GetGroupList)

	s.engine.Any("/get_group_info", s.GetGroupInfo)
	s.engine.Any("/get_group_info_async", s.GetGroupInfo)

	s.engine.Any("/get_group_member_list", s.GetGroupMemberList)
	s.engine.Any("/get_group_member_list_async", s.GetGroupMemberList)

	s.engine.Any("/get_group_member_info", s.GetGroupMemberInfo)
	s.engine.Any("/get_group_member_info_async", s.GetGroupMemberInfo)

	s.engine.Any("/send_msg", s.SendMessage)
	s.engine.Any("/send_msg_async", s.SendMessage)

	s.engine.Any("/send_private_msg", s.SendPrivateMessage)
	s.engine.Any("/send_private_msg_async", s.SendPrivateMessage)

	s.engine.Any("/send_group_msg", s.SendGroupMessage)
	s.engine.Any("/send_group_msg_async", s.SendGroupMessage)

	s.engine.Any("/send_group_forward_msg", s.SendGroupForwardMessage)
	s.engine.Any("/send_group_forward_msg_async", s.SendGroupForwardMessage)

	s.engine.Any("/delete_msg", s.DeleteMessage)
	s.engine.Any("/delete_msg_async", s.DeleteMessage)

	s.engine.Any("/set_friend_add_request", s.ProcessFriendRequest)
	s.engine.Any("/set_friend_add_request_async", s.ProcessFriendRequest)

	s.engine.Any("/set_group_add_request", s.ProcessGroupRequest)
	s.engine.Any("/set_group_add_request_async", s.ProcessGroupRequest)

	s.engine.Any("/set_group_card", s.SetGroupCard)
	s.engine.Any("/set_group_card_async", s.SetGroupCard)

	s.engine.Any("/set_group_special_title", s.SetSpecialTitle)
	s.engine.Any("/set_group_special_title_async", s.SetSpecialTitle)

	s.engine.Any("/set_group_kick", s.SetGroupKick)
	s.engine.Any("/set_group_kick_async", s.SetGroupKick)

	s.engine.Any("/set_group_ban", s.SetGroupBan)
	s.engine.Any("/set_group_ban_async", s.SetGroupBan)

	s.engine.Any("/set_group_whole_ban", s.SetWholeBan)
	s.engine.Any("/set_group_whole_ban_async", s.SetWholeBan)

	s.engine.Any("/set_group_name", s.SetGroupName)
	s.engine.Any("/set_group_name_async", s.SetGroupName)

	s.engine.Any("/set_group_leave", s.SetGroupLeave)
	s.engine.Any("/set_group_leave_async", s.SetGroupLeave)

	s.engine.Any("/get_image", s.GetImage)

	s.engine.Any("/get_forward_msg", s.GetForwardMessage)

	s.engine.Any("/get_group_msg", s.GetGroupMessage)

	s.engine.Any("/get_group_honor_info", s.GetGroupHonorInfo)

	s.engine.Any("/can_send_image", s.CanSendImage)
	s.engine.Any("/can_send_image_async", s.CanSendImage)

	s.engine.Any("/can_send_record", s.CanSendRecord)
	s.engine.Any("/can_send_record_async", s.CanSendRecord)

	s.engine.Any("/get_status", s.GetStatus)
	s.engine.Any("/get_status_async", s.GetStatus)

	s.engine.Any("/get_version_info", s.GetVersionInfo)
	s.engine.Any("/get_version_info_async", s.GetVersionInfo)

	s.engine.Any("/.handle_quick_operation", s.HandleQuickOperation)

	go func() {
		log.Infof("CQ HTTP 服务器已启动: %v", addr)
		err := s.engine.Run(addr)
		if err != nil {
			log.Error(err)
			log.Infof("请检查端口是否被占用.")
			time.Sleep(time.Second * 5)
			os.Exit(1)
		}
	}()
}

func NewHttpClient() *httpClient {
	return &httpClient{}
}

func (c *httpClient) Run(addr, secret string, timeout int32, bot *coolq.CQBot) {
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

func (c *httpClient) onBotPushEvent(m coolq.MSG) {
	var res string
	err := gout.POST(c.addr).SetJSON(m).BindBody(&res).SetHeader(func() gout.H {
		h := gout.H{
			"X-Self-ID":  c.bot.Client.Uin,
			"User-Agent": "CQHttp/4.15.0",
		}
		if c.secret != "" {
			mac := hmac.New(sha1.New, []byte(c.secret))
			mac.Write([]byte(m.ToJson()))
			h["X-Signature"] = "sha1=" + hex.EncodeToString(mac.Sum(nil))
		}
		return h
	}()).SetTimeout(time.Second * time.Duration(c.timeout)).Do()
	if err != nil {
		log.Warnf("上报Event数据到 %v 失败: %v", c.addr, err)
		return
	}
	if gjson.Valid(res) {
		c.bot.CQHandleQuickOperation(gjson.Parse(m.ToJson()), gjson.Parse(res))
	}
}

func (s *httpServer) GetLoginInfo(c *gin.Context) {
	c.JSON(200, s.bot.CQGetLoginInfo())
}

func (s *httpServer) GetFriendList(c *gin.Context) {
	c.JSON(200, s.bot.CQGetFriendList())
}

func (s *httpServer) GetGroupList(c *gin.Context) {
	nc := getParamOrDefault(c, "no_cache", "false")
	c.JSON(200, s.bot.CQGetGroupList(nc == "true"))
}

func (s *httpServer) GetGroupInfo(c *gin.Context) {
	gid, _ := strconv.ParseInt(getParam(c, "group_id"), 10, 64)
	c.JSON(200, s.bot.CQGetGroupInfo(gid))
}

func (s *httpServer) GetGroupMemberList(c *gin.Context) {
	gid, _ := strconv.ParseInt(getParam(c, "group_id"), 10, 64)
	c.JSON(200, s.bot.CQGetGroupMemberList(gid))
}

func (s *httpServer) GetGroupMemberInfo(c *gin.Context) {
	gid, _ := strconv.ParseInt(getParam(c, "group_id"), 10, 64)
	uid, _ := strconv.ParseInt(getParam(c, "user_id"), 10, 64)
	nc := getParamOrDefault(c, "no_cache", "false")
	c.JSON(200, s.bot.CQGetGroupMemberInfo(gid, uid, nc == "true"))
}

func (s *httpServer) SendMessage(c *gin.Context) {
	if getParam(c, "message_type") == "private" {
		s.SendPrivateMessage(c)
		return
	}
	if getParam(c, "message_type") == "group" {
		s.SendGroupMessage(c)
		return
	}
	if getParam(c, "group_id") != "" {
		s.SendGroupMessage(c)
		return
	}
	if getParam(c, "user_id") != "" {
		s.SendPrivateMessage(c)
	}
}

func (s *httpServer) SendPrivateMessage(c *gin.Context) {
	uid, _ := strconv.ParseInt(getParam(c, "user_id"), 10, 64)
	msg, t := getParamWithType(c, "message")
	autoEscape := global.EnsureBool(getParam(c, "auto_escape"), false)
	if t == gjson.JSON {
		c.JSON(200, s.bot.CQSendPrivateMessage(uid, gjson.Parse(msg), autoEscape))
		return
	}
	c.JSON(200, s.bot.CQSendPrivateMessage(uid, msg, autoEscape))
}

func (s *httpServer) SendGroupMessage(c *gin.Context) {
	gid, _ := strconv.ParseInt(getParam(c, "group_id"), 10, 64)
	msg, t := getParamWithType(c, "message")
	autoEscape := global.EnsureBool(getParam(c, "auto_escape"), false)
	if t == gjson.JSON {
		c.JSON(200, s.bot.CQSendGroupMessage(gid, gjson.Parse(msg), autoEscape))
		return
	}
	c.JSON(200, s.bot.CQSendGroupMessage(gid, msg, autoEscape))
}

func (s *httpServer) SendGroupForwardMessage(c *gin.Context) {
	gid, _ := strconv.ParseInt(getParam(c, "group_id"), 10, 64)
	msg := getParam(c, "messages")
	c.JSON(200, s.bot.CQSendGroupForwardMessage(gid, gjson.Parse(msg)))
}

func (s *httpServer) GetImage(c *gin.Context) {
	file := getParam(c, "file")
	c.JSON(200, s.bot.CQGetImage(file))
}

func (s *httpServer) GetGroupMessage(c *gin.Context) {
	mid, _ := strconv.ParseInt(getParam(c, "message_id"), 10, 32)
	c.JSON(200, s.bot.CQGetGroupMessage(int32(mid)))
}

func (s *httpServer) GetGroupHonorInfo(c *gin.Context) {
	gid, _ := strconv.ParseInt(getParam(c, "group_id"), 10, 64)
	c.JSON(200, s.bot.CQGetGroupHonorInfo(gid, getParam(c, "type")))
}

func (s *httpServer) ProcessFriendRequest(c *gin.Context) {
	flag := getParam(c, "flag")
	approve := getParamOrDefault(c, "approve", "true")
	c.JSON(200, s.bot.CQProcessFriendRequest(flag, approve == "true"))
}

func (s *httpServer) ProcessGroupRequest(c *gin.Context) {
	flag := getParam(c, "flag")
	subType := getParam(c, "sub_type")
	if subType == "" {
		subType = getParam(c, "type")
	}
	approve := getParamOrDefault(c, "approve", "true")
	c.JSON(200, s.bot.CQProcessGroupRequest(flag, subType, getParam(c, "reason"), approve == "true"))
}

func (s *httpServer) SetGroupCard(c *gin.Context) {
	gid, _ := strconv.ParseInt(getParam(c, "group_id"), 10, 64)
	uid, _ := strconv.ParseInt(getParam(c, "user_id"), 10, 64)
	c.JSON(200, s.bot.CQSetGroupCard(gid, uid, getParam(c, "card")))
}

func (s *httpServer) SetSpecialTitle(c *gin.Context) {
	gid, _ := strconv.ParseInt(getParam(c, "group_id"), 10, 64)
	uid, _ := strconv.ParseInt(getParam(c, "user_id"), 10, 64)
	c.JSON(200, s.bot.CQSetGroupSpecialTitle(gid, uid, getParam(c, "special_title")))
}

func (s *httpServer) SetGroupKick(c *gin.Context) {
	gid, _ := strconv.ParseInt(getParam(c, "group_id"), 10, 64)
	uid, _ := strconv.ParseInt(getParam(c, "user_id"), 10, 64)
	msg := getParam(c, "message")
	c.JSON(200, s.bot.CQSetGroupKick(gid, uid, msg))
}

func (s *httpServer) SetGroupBan(c *gin.Context) {
	gid, _ := strconv.ParseInt(getParam(c, "group_id"), 10, 64)
	uid, _ := strconv.ParseInt(getParam(c, "user_id"), 10, 64)
	i, _ := strconv.ParseInt(getParamOrDefault(c, "duration", "1800"), 10, 64)
	c.JSON(200, s.bot.CQSetGroupBan(gid, uid, uint32(i)))
}

func (s *httpServer) SetWholeBan(c *gin.Context) {
	gid, _ := strconv.ParseInt(getParam(c, "group_id"), 10, 64)
	c.JSON(200, s.bot.CQSetGroupWholeBan(gid, getParamOrDefault(c, "enable", "true") == "true"))
}

func (s *httpServer) SetGroupName(c *gin.Context) {
	gid, _ := strconv.ParseInt(getParam(c, "group_id"), 10, 64)
	c.JSON(200, s.bot.CQSetGroupName(gid, getParam(c, "name")))
}

func (s *httpServer) SetGroupLeave(c *gin.Context) {
	gid, _ := strconv.ParseInt(getParam(c, "group_id"), 10, 64)
	c.JSON(200, s.bot.CQSetGroupLeave(gid))
}

func (s *httpServer) GetForwardMessage(c *gin.Context) {
	resId := getParam(c, "message_id")
	c.JSON(200, s.bot.CQGetForwardMessage(resId))
}

func (s *httpServer) DeleteMessage(c *gin.Context) {
	mid, _ := strconv.ParseInt(getParam(c, "message_id"), 10, 32)
	c.JSON(200, s.bot.CQDeleteMessage(int32(mid)))
}

func (s *httpServer) CanSendImage(c *gin.Context) {
	c.JSON(200, s.bot.CQCanSendImage())
}

func (s *httpServer) CanSendRecord(c *gin.Context) {
	c.JSON(200, s.bot.CQCanSendRecord())
}

func (s *httpServer) GetStatus(c *gin.Context) {
	c.JSON(200, s.bot.CQGetStatus())
}

func (s *httpServer) GetVersionInfo(c *gin.Context) {
	c.JSON(200, s.bot.CQGetVersionInfo())
}

func (s *httpServer) HandleQuickOperation(c *gin.Context) {
	if c.Request.Method != "POST" {
		c.AbortWithStatus(404)
		return
	}
	if i, ok := c.Get("json_body"); ok {
		body := i.(gjson.Result)
		c.JSON(200, s.bot.CQHandleQuickOperation(body.Get("context"), body.Get("operation")))
	}
}

func getParamOrDefault(c *gin.Context, k, def string) string {
	r := getParam(c, k)
	if r != "" {
		return r
	}
	return def
}


func getParam(c *gin.Context, k string) string {
	p, _ := getParamWithType(c, k)
	return p
}

func getParamWithType(c *gin.Context, k string) (string, gjson.Type) {
	if q := c.Query(k); q != "" {
		return q, gjson.Null
	}
	if c.Request.Method == "POST" {
		if h := c.Request.Header.Get("Content-Type"); h != "" {
			if strings.Contains(h, "application/x-www-form-urlencoded") {
				if p, ok := c.GetPostForm(k); ok {
					return p, gjson.Null
				}
			}
			if strings.Contains(h, "application/json") {
				if obj, ok := c.Get("json_body"); ok {
					res := obj.(gjson.Result).Get(k)
					if res.Exists() {
						switch res.Type {
						case gjson.JSON:
							return res.Raw, gjson.JSON
						case gjson.String:
							return res.Str, gjson.String
						case gjson.Number:
							return strconv.FormatInt(res.Int(), 10), gjson.Number // 似乎没有需要接受 float 类型的api
						case gjson.True:
							return "true", gjson.True
						case gjson.False:
							return "false", gjson.False
						}
					}
				}
			}
		}
	}
	return "", gjson.Null
}
