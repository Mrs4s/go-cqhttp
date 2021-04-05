package server

import (
	"strings"

	"github.com/Mrs4s/go-cqhttp/coolq"
	"github.com/Mrs4s/go-cqhttp/global"

	"github.com/tidwall/gjson"
)

type resultGetter interface {
	Get(string) gjson.Result
}

type handler func(action string, p resultGetter) coolq.MSG

type apiCaller struct {
	bot      *coolq.CQBot
	handlers []handler
}

func getLoginInfo(bot *coolq.CQBot, _ resultGetter) coolq.MSG {
	return bot.CQGetLoginInfo()
}

func getFriendList(bot *coolq.CQBot, _ resultGetter) coolq.MSG {
	return bot.CQGetFriendList()
}

func getGroupList(bot *coolq.CQBot, p resultGetter) coolq.MSG {
	return bot.CQGetGroupList(p.Get("no_cache").Bool())
}

func getGroupInfo(bot *coolq.CQBot, p resultGetter) coolq.MSG {
	return bot.CQGetGroupInfo(p.Get("group_id").Int(), p.Get("no_cache").Bool())
}

func getGroupMemberList(bot *coolq.CQBot, p resultGetter) coolq.MSG {
	return bot.CQGetGroupMemberList(p.Get("group_id").Int(), p.Get("no_cache").Bool())
}

func getGroupMemberInfo(bot *coolq.CQBot, p resultGetter) coolq.MSG {
	return bot.CQGetGroupMemberInfo(
		p.Get("group_id").Int(), p.Get("user_id").Int(), p.Get("no_cache").Bool(),
	)
}

func sendMSG(bot *coolq.CQBot, p resultGetter) coolq.MSG {
	autoEscape := global.EnsureBool(p.Get("auto_escape"), false)
	if p.Get("message_type").Str == "private" {
		return bot.CQSendPrivateMessage(p.Get("user_id").Int(), p.Get("group_id").Int(), p.Get("message"), autoEscape)
	}
	if p.Get("message_type").Str == "group" {
		return bot.CQSendGroupMessage(p.Get("group_id").Int(), p.Get("message"), autoEscape)
	}
	if p.Get("user_id").Int() != 0 {
		return bot.CQSendPrivateMessage(p.Get("user_id").Int(), p.Get("group_id").Int(), p.Get("message"), autoEscape)
	}
	if p.Get("group_id").Int() != 0 {
		return bot.CQSendGroupMessage(p.Get("group_id").Int(), p.Get("message"), autoEscape)
	}
	return coolq.MSG{}
}

func sendGroupMSG(bot *coolq.CQBot, p resultGetter) coolq.MSG {
	return bot.CQSendGroupMessage(p.Get("group_id").Int(), p.Get("message"),
		global.EnsureBool(p.Get("auto_escape"), false))
}

func sendGroupForwardMSG(bot *coolq.CQBot, p resultGetter) coolq.MSG {
	return bot.CQSendGroupForwardMessage(p.Get("group_id").Int(), p.Get("messages"))
}

func sendPrivateMSG(bot *coolq.CQBot, p resultGetter) coolq.MSG {
	return bot.CQSendPrivateMessage(p.Get("user_id").Int(), p.Get("group_id").Int(), p.Get("message"),
		global.EnsureBool(p.Get("auto_escape"), false))
}

func deleteMSG(bot *coolq.CQBot, p resultGetter) coolq.MSG {
	return bot.CQDeleteMessage(int32(p.Get("message_id").Int()))
}

func setFriendAddRequest(bot *coolq.CQBot, p resultGetter) coolq.MSG {
	apr := true
	if p.Get("approve").Exists() {
		apr = p.Get("approve").Bool()
	}
	return bot.CQProcessFriendRequest(p.Get("flag").Str, apr)
}

func setGroupAddRequest(bot *coolq.CQBot, p resultGetter) coolq.MSG {
	subType := p.Get("sub_type").Str
	apr := true
	if subType == "" {
		subType = p.Get("type").Str
	}
	if p.Get("approve").Exists() {
		apr = p.Get("approve").Bool()
	}
	return bot.CQProcessGroupRequest(p.Get("flag").Str, subType, p.Get("reason").Str, apr)
}

func setGroupCard(bot *coolq.CQBot, p resultGetter) coolq.MSG {
	return bot.CQSetGroupCard(p.Get("group_id").Int(), p.Get("user_id").Int(), p.Get("card").Str)
}

func setGroupSpecialTitle(bot *coolq.CQBot, p resultGetter) coolq.MSG {
	return bot.CQSetGroupSpecialTitle(p.Get("group_id").Int(), p.Get("user_id").Int(), p.Get("special_title").Str)
}

func setGroupKick(bot *coolq.CQBot, p resultGetter) coolq.MSG {
	return bot.CQSetGroupKick(p.Get("group_id").Int(), p.Get("user_id").Int(), p.Get("message").Str, p.Get("reject_add_request").Bool())
}

func setGroupBan(bot *coolq.CQBot, p resultGetter) coolq.MSG {
	return bot.CQSetGroupBan(p.Get("group_id").Int(), p.Get("user_id").Int(), func() uint32 {
		if p.Get("duration").Exists() {
			return uint32(p.Get("duration").Int())
		}
		return 1800
	}())
}

func setGroupWholeBan(bot *coolq.CQBot, p resultGetter) coolq.MSG {
	return bot.CQSetGroupWholeBan(p.Get("group_id").Int(), func() bool {
		if p.Get("enable").Exists() {
			return p.Get("enable").Bool()
		}
		return true
	}())
}

func setGroupName(bot *coolq.CQBot, p resultGetter) coolq.MSG {
	return bot.CQSetGroupName(p.Get("group_id").Int(), p.Get("group_name").Str)
}

func setGroupAdmin(bot *coolq.CQBot, p resultGetter) coolq.MSG {
	return bot.CQSetGroupAdmin(p.Get("group_id").Int(), p.Get("user_id").Int(), func() bool {
		if p.Get("enable").Exists() {
			return p.Get("enable").Bool()
		}
		return true
	}())
}

func sendGroupNotice(bot *coolq.CQBot, p resultGetter) coolq.MSG {
	return bot.CQSetGroupMemo(p.Get("group_id").Int(), p.Get("content").Str)
}

func setGroupLeave(bot *coolq.CQBot, p resultGetter) coolq.MSG {
	return bot.CQSetGroupLeave(p.Get("group_id").Int())
}

func getImage(bot *coolq.CQBot, p resultGetter) coolq.MSG {
	return bot.CQGetImage(p.Get("file").Str)
}

func getForwardMSG(bot *coolq.CQBot, p resultGetter) coolq.MSG {
	id := p.Get("message_id").Str
	if id == "" {
		id = p.Get("id").Str
	}
	return bot.CQGetForwardMessage(id)
}

func getMSG(bot *coolq.CQBot, p resultGetter) coolq.MSG {
	return bot.CQGetMessage(int32(p.Get("message_id").Int()))
}

func downloadFile(bot *coolq.CQBot, p resultGetter) coolq.MSG {
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
}

func getGroupHonorInfo(bot *coolq.CQBot, p resultGetter) coolq.MSG {
	return bot.CQGetGroupHonorInfo(p.Get("group_id").Int(), p.Get("type").Str)
}

func setRestart(_ *coolq.CQBot, _ resultGetter) coolq.MSG {
	/*
		var delay int64
		delay = p.Get("delay").Int()
		if delay < 0 {
			delay = 0
		}
		defer func(delay int64) {
			time.Sleep(time.Duration(delay) * time.Millisecond)
			Restart <- struct{}{}
		}(delay)
	*/
	return coolq.MSG{"data": nil, "retcode": 99, "msg": "restart un-supported now", "wording": "restart函数暂不兼容", "status": "failed"}
}

func canSendImage(bot *coolq.CQBot, _ resultGetter) coolq.MSG {
	return bot.CQCanSendImage()
}

func canSendRecord(bot *coolq.CQBot, _ resultGetter) coolq.MSG {
	return bot.CQCanSendRecord()
}

func getStrangerInfo(bot *coolq.CQBot, p resultGetter) coolq.MSG {
	return bot.CQGetStrangerInfo(p.Get("user_id").Int())
}

func getStatus(bot *coolq.CQBot, _ resultGetter) coolq.MSG {
	return bot.CQGetStatus()
}

func getVersionInfo(bot *coolq.CQBot, _ resultGetter) coolq.MSG {
	return bot.CQGetVersionInfo()
}

func getGroupSystemMSG(bot *coolq.CQBot, _ resultGetter) coolq.MSG {
	return bot.CQGetGroupSystemMessages()
}

func getGroupFileSystemInfo(bot *coolq.CQBot, p resultGetter) coolq.MSG {
	return bot.CQGetGroupFileSystemInfo(p.Get("group_id").Int())
}

func getGroupRootFiles(bot *coolq.CQBot, p resultGetter) coolq.MSG {
	return bot.CQGetGroupRootFiles(p.Get("group_id").Int())
}

func getGroupFilesByFolder(bot *coolq.CQBot, p resultGetter) coolq.MSG {
	return bot.CQGetGroupFilesByFolderID(p.Get("group_id").Int(), p.Get("folder_id").Str)
}

func getGroupFileURL(bot *coolq.CQBot, p resultGetter) coolq.MSG {
	return bot.CQGetGroupFileURL(p.Get("group_id").Int(), p.Get("file_id").Str, int32(p.Get("busid").Int()))
}

func uploadGroupFile(bot *coolq.CQBot, p resultGetter) coolq.MSG {
	return bot.CQUploadGroupFile(p.Get("group_id").Int(), p.Get("file").Str, p.Get("name").Str, p.Get("folder").Str)
}

func getGroupMsgHistory(bot *coolq.CQBot, p resultGetter) coolq.MSG {
	return bot.CQGetGroupMessageHistory(p.Get("group_id").Int(), p.Get("message_seq").Int())
}

func getVipInfo(bot *coolq.CQBot, p resultGetter) coolq.MSG {
	return bot.CQGetVipInfo(p.Get("user_id").Int())
}

func reloadEventFilter(_ *coolq.CQBot, p resultGetter) coolq.MSG {
	addFilter(p.Get("file").String())
	return coolq.OK(nil)
}

func getGroupAtAllRemain(bot *coolq.CQBot, p resultGetter) coolq.MSG {
	return bot.CQGetAtAllRemain(p.Get("group_id").Int())
}

func ocrImage(bot *coolq.CQBot, p resultGetter) coolq.MSG {
	return bot.CQOcrImage(p.Get("image").Str)
}

func getOnlineClients(bot *coolq.CQBot, p resultGetter) coolq.MSG {
	return bot.CQGetOnlineClients(p.Get("no_cache").Bool())
}

func getWordSlices(bot *coolq.CQBot, p resultGetter) coolq.MSG {
	return bot.CQGetWordSlices(p.Get("content").Str)
}

func setGroupPortrait(bot *coolq.CQBot, p resultGetter) coolq.MSG {
	return bot.CQSetGroupPortrait(p.Get("group_id").Int(), p.Get("file").String(), p.Get("cache").String())
}

func setEssenceMSG(bot *coolq.CQBot, p resultGetter) coolq.MSG {
	return bot.CQSetEssenceMessage(int32(p.Get("message_id").Int()))
}

func deleteEssenceMSG(bot *coolq.CQBot, p resultGetter) coolq.MSG {
	return bot.CQDeleteEssenceMessage(int32(p.Get("message_id").Int()))
}

func getEssenceMsgList(bot *coolq.CQBot, p resultGetter) coolq.MSG {
	return bot.CQGetEssenceMessageList(p.Get("group_id").Int())
}

func checkURLSafely(bot *coolq.CQBot, p resultGetter) coolq.MSG {
	return bot.CQCheckURLSafely(p.Get("url").String())
}

func setGroupAnonymousBan(bot *coolq.CQBot, p resultGetter) coolq.MSG {
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
}

func handleQuickOperation(bot *coolq.CQBot, p resultGetter) coolq.MSG {
	return bot.CQHandleQuickOperation(p.Get("context"), p.Get("operation"))
}

// API 是go-cqhttp当前支持的所有api的映射表
var API = map[string]func(*coolq.CQBot, resultGetter) coolq.MSG{
	"get_login_info":             getLoginInfo,
	"get_friend_list":            getFriendList,
	"get_group_list":             getGroupList,
	"get_group_info":             getGroupInfo,
	"get_group_member_list":      getGroupMemberList,
	"get_group_member_info":      getGroupMemberInfo,
	"send_msg":                   sendMSG,
	"send_group_msg":             sendGroupMSG,
	"send_group_forward_msg":     sendGroupForwardMSG,
	"send_private_msg":           sendPrivateMSG,
	"delete_msg":                 deleteMSG,
	"set_friend_add_request":     setFriendAddRequest,
	"set_group_add_request":      setGroupAddRequest,
	"set_group_card":             setGroupCard,
	"set_group_special_title":    setGroupSpecialTitle,
	"set_group_kick":             setGroupKick,
	"set_group_ban":              setGroupBan,
	"set_group_whole_ban":        setGroupWholeBan,
	"set_group_name":             setGroupName,
	"set_group_admin":            setGroupAdmin,
	"_send_group_notice":         sendGroupNotice,
	"set_group_leave":            setGroupLeave,
	"get_image":                  getImage,
	"get_forward_msg":            getForwardMSG,
	"get_msg":                    getMSG,
	"download_file":              downloadFile,
	"get_group_honor_info":       getGroupHonorInfo,
	"set_restart":                setRestart,
	"can_send_image":             canSendImage,
	"can_send_record":            canSendRecord,
	"get_stranger_info":          getStrangerInfo,
	"get_status":                 getStatus,
	"get_version_info":           getVersionInfo,
	"get_group_system_msg":       getGroupSystemMSG,
	"get_group_file_system_info": getGroupFileSystemInfo,
	"get_group_root_files":       getGroupRootFiles,
	"get_group_files_by_folder":  getGroupFilesByFolder,
	"get_group_file_url":         getGroupFileURL,
	"upload_group_file":          uploadGroupFile,
	"get_group_msg_history":      getGroupMsgHistory,
	"_get_vip_info":              getVipInfo,
	"reload_event_filter":        reloadEventFilter,
	".ocr_image":                 ocrImage,
	"ocr_image":                  ocrImage,
	"get_group_at_all_remain":    getGroupAtAllRemain,
	"get_online_clients":         getOnlineClients,
	".get_word_slices":           getWordSlices,
	"set_group_portrait":         setGroupPortrait,
	"set_essence_msg":            setEssenceMSG,
	"delete_essence_msg":         deleteEssenceMSG,
	"get_essence_msg_list":       getEssenceMsgList,
	"check_url_safely":           checkURLSafely,
	"set_group_anonymous_ban":    setGroupAnonymousBan,
	".handle_quick_operation":    handleQuickOperation,
}

func (api *apiCaller) callAPI(action string, p resultGetter) coolq.MSG {
	for _, fn := range api.handlers {
		if ret := fn(action, p); ret != nil {
			return ret
		}
	}
	if f, ok := API[action]; ok {
		return f(api.bot, p)
	}
	return coolq.Failed(404, "API_NOT_FOUND", "API不存在")
}

func (api *apiCaller) use(middlewares ...handler) {
	api.handlers = append(api.handlers, middlewares...)
}

func newAPICaller(bot *coolq.CQBot) *apiCaller {
	return &apiCaller{
		bot:      bot,
		handlers: []handler{},
	}
}
