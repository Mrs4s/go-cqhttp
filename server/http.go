package server

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/guonaihong/gout/dataflow"

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
	HTTP   *http.Server
}

type httpClient struct {
	bot     *coolq.CQBot
	secret  string
	addr    string
	timeout int32
}

var cqHTTPServer = &httpServer{}

// Debug 是否启用Debug模式
var Debug = false

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

func newHTTPClient() *httpClient {
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
			_, err := mac.Write([]byte(m.ToJSON()))
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
		log.Warnf("上报Event数据 %v 到 %v 失败: %v", m.ToJSON(), c.addr, err)
		return
	}
	log.Debugf("上报Event数据 %v 到 %v", m.ToJSON(), c.addr)
	if gjson.Valid(res) {
		c.bot.CQHandleQuickOperation(gjson.Parse(m.ToJSON()), gjson.Parse(res))
	}
}

func (s *httpServer) HandleActions(c *gin.Context) {
	global.RateLimit(context.Background())
	action := strings.ReplaceAll(c.Param("action"), "_async", "")
	log.Debugf("HTTPServer接收到API调用: %v", action)
	if f, ok := httpAPI[action]; ok {
		f(s, c)
	} else {
		c.JSON(200, coolq.Failed(404))
	}
}

// GetLoginInfo 获取登录号信息
func GetLoginInfo(s *httpServer, c *gin.Context) {
	c.JSON(200, s.bot.CQGetLoginInfo())
}

// GetFriendList 获取好友列表
func GetFriendList(s *httpServer, c *gin.Context) {
	c.JSON(200, s.bot.CQGetFriendList())
}

// GetGroupList 获取群列表
func GetGroupList(s *httpServer, c *gin.Context) {
	nc := getParamOrDefault(c, "no_cache", "false")
	c.JSON(200, s.bot.CQGetGroupList(nc == "true"))
}

// GetGroupInfo 获取群信息
func GetGroupInfo(s *httpServer, c *gin.Context) {
	gid, _ := strconv.ParseInt(getParam(c, "group_id"), 10, 64)
	nc := getParamOrDefault(c, "no_cache", "false")
	c.JSON(200, s.bot.CQGetGroupInfo(gid, nc == "true"))
}

// GetGroupMemberList 获取群成员列表
func GetGroupMemberList(s *httpServer, c *gin.Context) {
	gid, _ := strconv.ParseInt(getParam(c, "group_id"), 10, 64)
	nc := getParamOrDefault(c, "no_cache", "false")
	c.JSON(200, s.bot.CQGetGroupMemberList(gid, nc == "true"))
}

// GetGroupMemberInfo 获取群成员信息
func GetGroupMemberInfo(s *httpServer, c *gin.Context) {
	gid, _ := strconv.ParseInt(getParam(c, "group_id"), 10, 64)
	uid, _ := strconv.ParseInt(getParam(c, "user_id"), 10, 64)
	c.JSON(200, s.bot.CQGetGroupMemberInfo(gid, uid))
}

// GetGroupFileSystemInfo 扩展API-获取群文件系统信息
func GetGroupFileSystemInfo(s *httpServer, c *gin.Context) {
	gid, _ := strconv.ParseInt(getParam(c, "group_id"), 10, 64)
	c.JSON(200, s.bot.CQGetGroupFileSystemInfo(gid))
}

// GetGroupRootFiles 扩展API-获取群根目录文件列表
func GetGroupRootFiles(s *httpServer, c *gin.Context) {
	gid, _ := strconv.ParseInt(getParam(c, "group_id"), 10, 64)
	c.JSON(200, s.bot.CQGetGroupRootFiles(gid))
}

// GetGroupFilesByFolderID 扩展API-获取群子目录文件列表
func GetGroupFilesByFolderID(s *httpServer, c *gin.Context) {
	gid, _ := strconv.ParseInt(getParam(c, "group_id"), 10, 64)
	folderID := getParam(c, "folder_id")
	c.JSON(200, s.bot.CQGetGroupFilesByFolderID(gid, folderID))
}

// GetGroupFileURL 扩展API-获取群文件资源链接
func GetGroupFileURL(s *httpServer, c *gin.Context) {
	gid, _ := strconv.ParseInt(getParam(c, "group_id"), 10, 64)
	fid := getParam(c, "file_id")
	busid, _ := strconv.ParseInt(getParam(c, "busid"), 10, 32)
	c.JSON(200, s.bot.CQGetGroupFileURL(gid, fid, int32(busid)))
}

// UploadGroupFile 扩展API-上传群文件
func UploadGroupFile(s *httpServer, c *gin.Context) {
	gid, _ := strconv.ParseInt(getParam(c, "group_id"), 10, 64)
	c.JSON(200, s.bot.CQUploadGroupFile(gid, getParam(c, "file"), getParam(c, "name"), getParam(c, "folder")))
}

// SendMessage 发送消息
//
// https://git.io/JtwTQ
func SendMessage(s *httpServer, c *gin.Context) {
	if getParam(c, "message_type") == "private" {
		SendPrivateMessage(s, c)
		return
	}
	if getParam(c, "message_type") == "group" {
		SendGroupMessage(s, c)
		return
	}
	if getParam(c, "group_id") != "" {
		SendGroupMessage(s, c)
		return
	}
	if getParam(c, "user_id") != "" {
		SendPrivateMessage(s, c)
	}
}

// SendPrivateMessage 发送私聊消息
func SendPrivateMessage(s *httpServer, c *gin.Context) {
	uid, _ := strconv.ParseInt(getParam(c, "user_id"), 10, 64)
	msg, t := getParamWithType(c, "message")
	autoEscape := global.EnsureBool(getParam(c, "auto_escape"), false)
	if t == gjson.JSON {
		c.JSON(200, s.bot.CQSendPrivateMessage(uid, gjson.Parse(msg), autoEscape))
		return
	}
	c.JSON(200, s.bot.CQSendPrivateMessage(uid, msg, autoEscape))
}

// SendGroupMessage 发送群消息
func SendGroupMessage(s *httpServer, c *gin.Context) {
	gid, _ := strconv.ParseInt(getParam(c, "group_id"), 10, 64)
	msg, t := getParamWithType(c, "message")
	autoEscape := global.EnsureBool(getParam(c, "auto_escape"), false)
	if t == gjson.JSON {
		c.JSON(200, s.bot.CQSendGroupMessage(gid, gjson.Parse(msg), autoEscape))
		return
	}
	c.JSON(200, s.bot.CQSendGroupMessage(gid, msg, autoEscape))
}

// SendGroupForwardMessage 扩展API-发送合并转发(群)
func SendGroupForwardMessage(s *httpServer, c *gin.Context) {
	gid, _ := strconv.ParseInt(getParam(c, "group_id"), 10, 64)
	msg := getParam(c, "messages")
	c.JSON(200, s.bot.CQSendGroupForwardMessage(gid, gjson.Parse(msg)))
}

// GetImage 获取图片(修改自OneBot)
func GetImage(s *httpServer, c *gin.Context) {
	file := getParam(c, "file")
	c.JSON(200, s.bot.CQGetImage(file))
}

// GetMessage 获取消息
func GetMessage(s *httpServer, c *gin.Context) {
	mid, _ := strconv.ParseInt(getParam(c, "message_id"), 10, 32)
	c.JSON(200, s.bot.CQGetMessage(int32(mid)))
}

// GetGroupHonorInfo 获取群荣誉信息
func GetGroupHonorInfo(s *httpServer, c *gin.Context) {
	gid, _ := strconv.ParseInt(getParam(c, "group_id"), 10, 64)
	c.JSON(200, s.bot.CQGetGroupHonorInfo(gid, getParam(c, "type")))
}

// ProcessFriendRequest 处理加好友请求
func ProcessFriendRequest(s *httpServer, c *gin.Context) {
	flag := getParam(c, "flag")
	approve := getParamOrDefault(c, "approve", "true")
	c.JSON(200, s.bot.CQProcessFriendRequest(flag, approve == "true"))
}

// ProcessGroupRequest 处理加群请求／邀请
func ProcessGroupRequest(s *httpServer, c *gin.Context) {
	flag := getParam(c, "flag")
	subType := getParam(c, "sub_type")
	if subType == "" {
		subType = getParam(c, "type")
	}
	approve := getParamOrDefault(c, "approve", "true")
	c.JSON(200, s.bot.CQProcessGroupRequest(flag, subType, getParam(c, "reason"), approve == "true"))
}

// SetGroupCard 设置群名片(群备注)
func SetGroupCard(s *httpServer, c *gin.Context) {
	gid, _ := strconv.ParseInt(getParam(c, "group_id"), 10, 64)
	uid, _ := strconv.ParseInt(getParam(c, "user_id"), 10, 64)
	c.JSON(200, s.bot.CQSetGroupCard(gid, uid, getParam(c, "card")))
}

// SetSpecialTitle 设置群组专属头衔
func SetSpecialTitle(s *httpServer, c *gin.Context) {
	gid, _ := strconv.ParseInt(getParam(c, "group_id"), 10, 64)
	uid, _ := strconv.ParseInt(getParam(c, "user_id"), 10, 64)
	c.JSON(200, s.bot.CQSetGroupSpecialTitle(gid, uid, getParam(c, "special_title")))
}

// SetGroupKick 群组踢人
func SetGroupKick(s *httpServer, c *gin.Context) {
	gid, _ := strconv.ParseInt(getParam(c, "group_id"), 10, 64)
	uid, _ := strconv.ParseInt(getParam(c, "user_id"), 10, 64)
	msg := getParam(c, "message")
	block := getParamOrDefault(c, "reject_add_request", "false")
	c.JSON(200, s.bot.CQSetGroupKick(gid, uid, msg, block == "true"))
}

// SetGroupBan 群组单人禁言
func SetGroupBan(s *httpServer, c *gin.Context) {
	gid, _ := strconv.ParseInt(getParam(c, "group_id"), 10, 64)
	uid, _ := strconv.ParseInt(getParam(c, "user_id"), 10, 64)
	i, _ := strconv.ParseInt(getParamOrDefault(c, "duration", "1800"), 10, 64)
	c.JSON(200, s.bot.CQSetGroupBan(gid, uid, uint32(i)))
}

// SetWholeBan 群组全员禁言
func SetWholeBan(s *httpServer, c *gin.Context) {
	gid, _ := strconv.ParseInt(getParam(c, "group_id"), 10, 64)
	c.JSON(200, s.bot.CQSetGroupWholeBan(gid, getParamOrDefault(c, "enable", "true") == "true"))
}

// SetGroupName 设置群名
func SetGroupName(s *httpServer, c *gin.Context) {
	gid, _ := strconv.ParseInt(getParam(c, "group_id"), 10, 64)
	c.JSON(200, s.bot.CQSetGroupName(gid, getParam(c, "group_name")))
}

// SetGroupAdmin 群组设置管理员
func SetGroupAdmin(s *httpServer, c *gin.Context) {
	gid, _ := strconv.ParseInt(getParam(c, "group_id"), 10, 64)
	uid, _ := strconv.ParseInt(getParam(c, "user_id"), 10, 64)
	c.JSON(200, s.bot.CQSetGroupAdmin(gid, uid, getParamOrDefault(c, "enable", "true") == "true"))
}

// SendGroupNotice 扩展API-发送群公告
func SendGroupNotice(s *httpServer, c *gin.Context) {
	gid, _ := strconv.ParseInt(getParam(c, "group_id"), 10, 64)
	c.JSON(200, s.bot.CQSetGroupMemo(gid, getParam(c, "content")))
}

// SetGroupLeave 退出群组
func SetGroupLeave(s *httpServer, c *gin.Context) {
	gid, _ := strconv.ParseInt(getParam(c, "group_id"), 10, 64)
	c.JSON(200, s.bot.CQSetGroupLeave(gid))
}

// SetRestart 重启 OneBot 实现
//
// https://git.io/JtwkJ
func SetRestart(s *httpServer, c *gin.Context) {
	delay, _ := strconv.ParseInt(getParam(c, "delay"), 10, 64)
	c.JSON(200, coolq.MSG{"data": nil, "retcode": 0, "status": "async"})
	go func(delay int64) {
		time.Sleep(time.Duration(delay) * time.Millisecond)
		Restart <- struct{}{}
	}(delay)

}

// GetForwardMessage 获取合并转发消息
func GetForwardMessage(s *httpServer, c *gin.Context) {
	resID := getParam(c, "message_id")
	if resID == "" {
		resID = getParam(c, "id")
	}
	c.JSON(200, s.bot.CQGetForwardMessage(resID))
}

// GetGroupSystemMessage 扩展API-获取群文件系统消息
func GetGroupSystemMessage(s *httpServer, c *gin.Context) {
	c.JSON(200, s.bot.CQGetGroupSystemMessages())
}

// DeleteMessage 撤回消息
func DeleteMessage(s *httpServer, c *gin.Context) {
	mid, _ := strconv.ParseInt(getParam(c, "message_id"), 10, 32)
	c.JSON(200, s.bot.CQDeleteMessage(int32(mid)))
}

// CanSendImage 检查是否可以发送图片(此处永远返回true)
func CanSendImage(s *httpServer, c *gin.Context) {
	c.JSON(200, s.bot.CQCanSendImage())
}

// CanSendRecord 检查是否可以发送语音(此处永远返回true)
func CanSendRecord(s *httpServer, c *gin.Context) {
	c.JSON(200, s.bot.CQCanSendRecord())
}

// GetStatus 获取运行状态
func GetStatus(s *httpServer, c *gin.Context) {
	c.JSON(200, s.bot.CQGetStatus())
}

// GetVersionInfo 获取版本信息
func GetVersionInfo(s *httpServer, c *gin.Context) {
	c.JSON(200, s.bot.CQGetVersionInfo())
}

// ReloadEventFilter 扩展API-重载事件过滤器
func ReloadEventFilter(s *httpServer, c *gin.Context) {
	c.JSON(200, s.bot.CQReloadEventFilter())
}

// GetVipInfo 扩展API-获取VIP信息
func GetVipInfo(s *httpServer, c *gin.Context) {
	uid, _ := strconv.ParseInt(getParam(c, "user_id"), 10, 64)
	c.JSON(200, s.bot.CQGetVipInfo(uid))
}

// GetStrangerInfo 获取陌生人信息
func GetStrangerInfo(s *httpServer, c *gin.Context) {
	uid, _ := strconv.ParseInt(getParam(c, "user_id"), 10, 64)
	c.JSON(200, s.bot.CQGetStrangerInfo(uid))
}

// GetGroupAtAllRemain 扩展API-获取群 @全体成员 剩余次数
func GetGroupAtAllRemain(s *httpServer, c *gin.Context) {
	gid, _ := strconv.ParseInt(getParam(c, "group_id"), 10, 64)
	c.JSON(200, s.bot.CQGetAtAllRemain(gid))
}

// SetGroupAnonymousBan 群组匿名用户禁言
func SetGroupAnonymousBan(s *httpServer, c *gin.Context) {
	gid, _ := strconv.ParseInt(getParam(c, "group_id"), 10, 64)
	d, _ := strconv.ParseInt(getParam(c, "duration"), 10, 64)
	flag := getParam(c, "flag")
	if flag == "" {
		flag = getParam(c, "anonymous_flag")
	}
	if flag == "" {
		o := gjson.Parse(getParam(c, "anonymous"))
		flag = o.Get("flag").String()
	}
	c.JSON(200, s.bot.CQSetGroupAnonymousBan(gid, flag, int32(d)))
}

// GetGroupMessageHistory 获取群消息历史记录
func GetGroupMessageHistory(s *httpServer, c *gin.Context) {
	gid, _ := strconv.ParseInt(getParam(c, "group_id"), 10, 64)
	seq, _ := strconv.ParseInt(getParam(c, "message_seq"), 10, 64)
	c.JSON(200, s.bot.CQGetGroupMessageHistory(gid, seq))
}

// GetOnlineClients 扩展API-获取当前账号在线客户端列表
func GetOnlineClients(s *httpServer, c *gin.Context) {
	c.JSON(200, s.bot.CQGetOnlineClients(getParamOrDefault(c, "no_cache", "false") == "true"))
}

// HandleQuickOperation 隐藏API-对事件执行快速操作
func HandleQuickOperation(s *httpServer, c *gin.Context) {
	if c.Request.Method != "POST" {
		c.AbortWithStatus(404)
		return
	}
	if i, ok := c.Get("json_body"); ok {
		body := i.(gjson.Result)
		c.JSON(200, s.bot.CQHandleQuickOperation(body.Get("context"), body.Get("operation")))
	}
}

// DownloadFile 扩展API-下载文件到缓存目录
func DownloadFile(s *httpServer, c *gin.Context) {
	url := getParam(c, "url")
	tc, _ := strconv.Atoi(getParam(c, "thread_count"))
	h, t := getParamWithType(c, "headers")
	headers := map[string]string{}
	if t == gjson.Null || t == gjson.String {
		lines := strings.Split(h, "\r\n")
		for _, sub := range lines {
			str := strings.SplitN(sub, "=", 2)
			if len(str) == 2 {
				headers[str[0]] = str[1]
			}
		}
	}
	if t == gjson.JSON {
		arr := gjson.Parse(h)
		for _, sub := range arr.Array() {
			str := strings.SplitN(sub.String(), "=", 2)
			if len(str) == 2 {
				headers[str[0]] = str[1]
			}
		}
	}
	println(url, tc, h, t)
	c.JSON(200, s.bot.CQDownloadFile(url, headers, tc))
}

// OcrImage 扩展API-图片OCR
func OcrImage(s *httpServer, c *gin.Context) {
	img := getParam(c, "image")
	c.JSON(200, s.bot.CQOcrImage(img))
}

// GetWordSlices 隐藏API-获取中文分词
func GetWordSlices(s *httpServer, c *gin.Context) {
	content := getParam(c, "content")
	c.JSON(200, s.bot.CQGetWordSlices(content))
}

// SetGroupPortrait 扩展API-设置群头像
func SetGroupPortrait(s *httpServer, c *gin.Context) {
	gid, _ := strconv.ParseInt(getParam(c, "group_id"), 10, 64)
	file := getParam(c, "file")
	cache := getParam(c, "cache")
	c.JSON(200, s.bot.CQSetGroupPortrait(gid, file, cache))
}

// SetEssenceMsg 扩展API-设置精华消息
func SetEssenceMsg(s *httpServer, c *gin.Context) {
	mid, _ := strconv.ParseInt(getParam(c, "message_id"), 10, 64)
	c.JSON(200, s.bot.CQSetEssenceMessage(int32(mid)))
}

// DeleteEssenceMsg 扩展API-移出精华消息
func DeleteEssenceMsg(s *httpServer, c *gin.Context) {
	mid, _ := strconv.ParseInt(getParam(c, "message_id"), 10, 64)
	c.JSON(200, s.bot.CQDeleteEssenceMessage(int32(mid)))
}

// GetEssenceMsgList 扩展API-获取精华消息列表
func GetEssenceMsgList(s *httpServer, c *gin.Context) {
	gid, _ := strconv.ParseInt(getParam(c, "group_id"), 10, 64)
	c.JSON(200, s.bot.CQGetEssenceMessageList(gid))
}

// CheckURLSafely 扩展API-检查链接安全性
func CheckURLSafely(s *httpServer, c *gin.Context) {
	c.JSON(200, s.bot.CQCheckURLSafely(getParam(c, "url")))
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

var httpAPI = map[string]func(s *httpServer, c *gin.Context){
	"get_login_info":             GetLoginInfo,
	"get_friend_list":            GetFriendList,
	"get_group_list":             GetGroupList,
	"get_group_info":             GetGroupInfo,
	"get_group_member_list":      GetGroupMemberList,
	"get_group_member_info":      GetGroupMemberInfo,
	"get_group_file_system_info": GetGroupFileSystemInfo,
	"get_group_root_files":       GetGroupRootFiles,
	"get_group_files_by_folder":  GetGroupFilesByFolderID,
	"get_group_file_url":         GetGroupFileURL,
	"upload_group_file":          UploadGroupFile,
	"get_essence_msg_list":       GetEssenceMsgList,
	"send_msg":                   SendMessage,
	"send_group_msg":             SendGroupMessage,
	"send_group_forward_msg":     SendGroupForwardMessage,
	"send_private_msg":           SendPrivateMessage,
	"delete_msg":                 DeleteMessage,
	"delete_essence_msg":         DeleteEssenceMsg,
	"set_friend_add_request":     ProcessFriendRequest,
	"set_group_add_request":      ProcessGroupRequest,
	"set_group_card":             SetGroupCard,
	"set_group_special_title":    SetSpecialTitle,
	"set_group_kick":             SetGroupKick,
	"set_group_ban":              SetGroupBan,
	"set_group_whole_ban":        SetWholeBan,
	"set_group_name":             SetGroupName,
	"set_group_admin":            SetGroupAdmin,
	"set_essence_msg":            SetEssenceMsg,
	"set_restart":                SetRestart,
	"_send_group_notice":         SendGroupNotice,
	"set_group_leave":            SetGroupLeave,
	"get_image":                  GetImage,
	"get_forward_msg":            GetForwardMessage,
	"get_msg":                    GetMessage,
	"get_group_system_msg":       GetGroupSystemMessage,
	"get_group_honor_info":       GetGroupHonorInfo,
	"can_send_image":             CanSendImage,
	"can_send_record":            CanSendRecord,
	"get_status":                 GetStatus,
	"get_version_info":           GetVersionInfo,
	"_get_vip_info":              GetVipInfo,
	"get_stranger_info":          GetStrangerInfo,
	"reload_event_filter":        ReloadEventFilter,
	"set_group_portrait":         SetGroupPortrait,
	"set_group_anonymous_ban":    SetGroupAnonymousBan,
	"get_group_msg_history":      GetGroupMessageHistory,
	"check_url_safely":           CheckURLSafely,
	"download_file":              DownloadFile,
	".handle_quick_operation":    HandleQuickOperation,
	".ocr_image":                 OcrImage,
	"ocr_image":                  OcrImage,
	"get_group_at_all_remain":    GetGroupAtAllRemain,
	"get_online_clients":         GetOnlineClients,
	".get_word_slices":           GetWordSlices,
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
