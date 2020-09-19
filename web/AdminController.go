package web

// 此Controller用于 需要鉴权 才能访问的cgi
import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/Mrs4s/go-cqhttp/global"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"html/template"
	"net/http"
	"os"
	"strconv"
	"time"
)

// admin 子站的 路由映射
var HttpuriAdmin = map[string]func(s *webServer, c *gin.Context){
	"index":               AdminIndex,
	"config_json":         AdminConfigJson,
	"config":              AdminConfig,
	"debug":               AdminDebug,
	"log":                 AdminLog,
	"friend_list":         AdminFriendList,
	"group_list":          AdminGroupList,
	"get_log":             AdmingetLogs,
	"do_friend_list":      AdminDoFriendList,
	"do_group_list":       AdminDoGroupList,
	"do_config_base":      AdminDoConfigBase,
	"do_config_http":      AdminDoConfigHttp,
	"do_config_ws":        AdminDoConfigWs,
	"do_config_reverse":   AdminDoConfigReverse,
	"do_config_json":      AdminDoConfigJson,
	"do_leave_group":      AdminDoLeaveGroup,
	"send_group_msg":      AdminSendGroupMsg,
	"do_group_msg_send":   AdminDoGoupMsgSend,
	"do_del_friend":       AdminDoDelFriend,
	"send_private_msg":    AdminSendPrivateMsg,
	"do_send_private_msg": AdminDoPrivateMsgSend,
	"restart":             AdminRestart,
	"do_restart":          AdminDoRestart,
	"web_write":           AdminWebWrite,
	"do_web_write":        AdminDoWebWrite,
	"restart_docker":      AdminRestartDocker,
	"do_restart_docker":   AdminDoRestartDocker,
}

// 首页
func AdminIndex(s *webServer, c *gin.Context) {
	c.HTML(http.StatusOK, "admin/index.html", gin.H{})
}

//json config
func AdminConfigJson(s *webServer, c *gin.Context) {
	conf := GetConf()
	data, _ := json.MarshalIndent(conf, "", "\t")
	c.HTML(http.StatusOK, "admin/config_json.html", gin.H{
		"json": template.HTML(data),
	})
}

// config
func AdminConfig(s *webServer, c *gin.Context) {
	conf := GetConf()
	var post string
	var secret string
	ws_reverse_servers := conf.ReverseServers[0]
	for k, v := range conf.HttpConfig.PostUrls {
		post = k
		secret = v
	}
	c.HTML(http.StatusOK, "admin/config.html", gin.H{
		"post":           post,
		"secret":         secret,
		"ReverseServers": ws_reverse_servers,
	})
}

// log
func AdminLog(s *webServer, c *gin.Context) {
	//读取当前日志
	LogsPath := getLogPath()
	Logs, _ := readLastLine(LogsPath, 10240)
	//conf := GetConf()
	//data, _ := json.MarshalIndent(conf, "", "\t")
	c.HTML(http.StatusOK, "admin/log.html", gin.H{
		"logs": template.HTML(Logs),
	})
}

// js获取log
func AdmingetLogs(s *webServer, c *gin.Context) {
	//读取当前日志
	LogsPath := getLogPath()
	Logs, err := readLastLine(LogsPath, 10240)
	if err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": "日志文件读取失败"})
	} else {
		c.JSON(200, gin.H{"code": 0, "logs": Logs})
	}
}

// api调试
func AdminDebug(s *webServer, c *gin.Context) {
	c.HTML(http.StatusOK, "admin/jump.html", gin.H{
		"url":     "/admin/index",
		"timeout": "3",
		"code":    0, //1为success,0为error
		"msg":     "开发中",
	})
	//c.JSON(http.StatusOK, gin.H{"code": -1, "msg": "开发中"})
}

// 好友列表html页面
func AdminFriendList(s *webServer, c *gin.Context) {
	c.HTML(http.StatusOK, "admin/friend_list.html", gin.H{})
}

// 群列表html页面
func AdminGroupList(s *webServer, c *gin.Context) {
	c.HTML(http.StatusOK, "admin/group_list.html", gin.H{})
}

// 好友列表js
func AdminDoFriendList(s *webServer, c *gin.Context) {
	c.JSON(200, s.bot.CQGetFriendList())
}

// 群列表js
func AdminDoGroupList(s *webServer, c *gin.Context) {
	c.JSON(200, s.bot.CQGetGroupList(false))
}

// 普通配置修改
func AdminDoConfigBase(s *webServer, c *gin.Context) {
	conf := GetConf()
	conf.Uin, _ = strconv.ParseInt(c.PostForm("uin"), 10, 64)
	conf.Password = c.PostForm("password")
	if c.PostForm("enable_db") == "true" {
		conf.EnableDB = true
	} else {
		conf.EnableDB = false
	}
	conf.AccessToken = c.PostForm("access_token")
	if err := conf.Save("config.json"); err != nil {
		//log.Fatalf("保存 config.json 时出现错误: %v", err)
		c.JSON(200, gin.H{"code": -1, "msg": "保存 config.json 时出现错误:" + fmt.Sprintf("%v", err)})
	} else {
		c.JSON(200, gin.H{"code": 0, "msg": "保存成功"})
	}
}

// http配置修改
func AdminDoConfigHttp(s *webServer, c *gin.Context) {
	conf := GetConf()
	p, _ := strconv.ParseUint(c.PostForm("port"), 10, 16)
	conf.HttpConfig.Port = uint16(p)
	conf.HttpConfig.Host = c.PostForm("host")
	if c.PostForm("enable") == "true" {
		conf.HttpConfig.Enabled = true
	} else {
		conf.HttpConfig.Enabled = false
	}
	t, _ := strconv.ParseInt(c.PostForm("timeout"), 10, 32)
	conf.HttpConfig.Timeout = int32(t)
	if c.PostForm("post_url") != "" {
		conf.HttpConfig.PostUrls[c.PostForm("post_url")] = c.PostForm("post_secret")
	}
	if err := conf.Save("config.json"); err != nil {
		//log.Fatalf("保存 config.json 时出现错误: %v", err)
		c.JSON(200, gin.H{"code": -1, "msg": "保存 config.json 时出现错误:" + fmt.Sprintf("%v", err)})
	} else {
		c.JSON(200, gin.H{"code": 0, "msg": "保存成功"})
	}
}

// ws配置修改
func AdminDoConfigWs(s *webServer, c *gin.Context) {
	conf := GetConf()
	p, _ := strconv.ParseUint(c.PostForm("port"), 10, 16)
	conf.WSConfig.Port = uint16(p)
	conf.WSConfig.Host = c.PostForm("host")
	if c.PostForm("enable") == "true" {
		conf.WSConfig.Enabled = true
	} else {
		conf.WSConfig.Enabled = false
	}
	if err := conf.Save("config.json"); err != nil {
		//log.Fatalf("保存 config.json 时出现错误: %v", err)
		c.JSON(200, gin.H{"code": -1, "msg": "保存 config.json 时出现错误:" + fmt.Sprintf("%v", err)})
	} else {
		c.JSON(200, gin.H{"code": 0, "msg": "保存成功"})
	}
}

// 反向ws配置修改
func AdminDoConfigReverse(s *webServer, c *gin.Context) {
	conf := GetConf()
	conf.ReverseServers[0].ReverseApiUrl = c.PostForm("reverse_api_url")
	conf.ReverseServers[0].ReverseUrl = c.PostForm("reverse_url")
	conf.ReverseServers[0].ReverseEventUrl = c.PostForm("reverse_event_url")
	t, _ := strconv.ParseUint(c.PostForm("reverse_reconnect_interval"), 10, 16)
	conf.ReverseServers[0].ReverseReconnectInterval = uint16(t)
	if c.PostForm("enable") == "true" {
		conf.ReverseServers[0].Enabled = true
	} else {
		conf.ReverseServers[0].Enabled = false
	}
	if err := conf.Save("config.json"); err != nil {
		//log.Fatalf("保存 config.json 时出现错误: %v", err)
		c.JSON(200, gin.H{"code": -1, "msg": "保存 config.json 时出现错误:" + fmt.Sprintf("%v", err)})
	} else {
		c.JSON(200, gin.H{"code": 0, "msg": "保存成功"})
	}
}

// 反向ws配置修改
func AdminDoConfigJson(s *webServer, c *gin.Context) {
	conf := GetConf()
	Json := c.PostForm("json")
	err := json.Unmarshal([]byte(Json), &conf)
	if err != nil {
		log.Warnf("尝试加载配置文件 %v 时出现错误: %v", "config.json", err)
		c.JSON(200, gin.H{"code": -1, "msg": "保存 config.json 时出现错误:" + fmt.Sprintf("%v", err)})
		return
	}
	if err := conf.Save("config.json"); err != nil {
		//log.Fatalf("保存 config.json 时出现错误: %v", err)
		c.JSON(200, gin.H{"code": -1, "msg": "保存 config.json 时出现错误:" + fmt.Sprintf("%v", err)})
	} else {
		c.JSON(200, gin.H{"code": 0, "msg": "保存成功"})
	}
}

// 退群
func AdminDoLeaveGroup(s *webServer, c *gin.Context) {
	gid, ext := c.GetQuery("gid")
	if !ext {
		c.HTML(http.StatusOK, "admin/jump.html", gin.H{
			"url":     "/admin/group_list",
			"timeout": "3",
			"code":    0, //1为success,0为error
			"msg":     "缺失参数gid/群号码",
		})
		c.Abort()
		return
	}
	groupId, _ := strconv.ParseInt(gid, 10, 64)
	rsp := s.bot.CQSetGroupLeave(groupId)
	if rsp["status"] == "ok" {
		c.HTML(http.StatusOK, "admin/jump.html", gin.H{
			"url":     "/admin/group_list",
			"timeout": 3,
			"code":    1, //1为success,0为error
			"msg":     "一键离群成功",
		})
		c.Abort()
		return
	} else {
		c.HTML(http.StatusOK, "admin/jump.html", gin.H{
			"url":     "/admin/group_list",
			"timeout": 3,
			"code":    0, //1为success,0为error
			"msg":     "一键离群失败",
		})
		c.Abort()
		return
	}
}

// 发送群消息html
func AdminSendGroupMsg(s *webServer, c *gin.Context) {
	gid, ext := c.GetQuery("gid")
	if !ext {
		c.HTML(http.StatusOK, "admin/jump.html", gin.H{
			"url":     "/admin/group_list",
			"timeout": "3",
			"code":    0, //1为success,0为error
			"msg":     "缺失参数gid/群号码",
		})
		c.Abort()
		return
	}
	groupId, _ := strconv.ParseInt(gid, 10, 64)
	rsp := s.bot.CQGetGroupInfo(groupId)
	if rsp["status"] != "ok" {
		c.HTML(http.StatusOK, "admin/jump.html", gin.H{
			"url":     "/admin/group_list",
			"timeout": 3,
			"code":    0, //1为success,0为error
			"msg":     "获取群信息失败",
		})
		c.Abort()
		return
	}
	Json := gjson.Parse(rsp.ToJson())
	groupName := Json.Get("data.group_name")
	c.HTML(http.StatusOK, "admin/send_group_msg.html", gin.H{
		"groupId":   groupId,
		"groupName": groupName,
	})
}

// 发送群消息
func AdminDoGoupMsgSend(s *webServer, c *gin.Context) {
	gid := c.PostForm("gid")
	if gid == "" {
		c.JSON(200, gin.H{
			"code": -1,
			"msg":  "缺失了gid/群号码",
		})
		c.Abort()
		return
	}
	msg := c.PostForm("msg")
	if msg == "" {
		c.JSON(200, gin.H{
			"code": -1,
			"msg":  "缺失了msg/消息内容",
		})
		c.Abort()
		return
	}
	groupId, _ := strconv.ParseInt(gid, 10, 64)
	rsp := s.bot.CQSendGroupMessage(groupId, msg, false)
	c.JSON(200, gin.H{
		"code": 0,
		"msg":  rsp,
	})
}

// 删好友
func AdminDoDelFriend(s *webServer, c *gin.Context) {
	//uid,ext:=c.GetQuery("uid")
	//if !ext{
	//	c.HTML(http.StatusOK, "jump.html", gin.H{
	//		"url":     "/admin/friend_list",
	//		"timeout": "3",
	//		"code":    0, //1为success,0为error
	//		"msg":     "缺失参数uid/qq号",
	//	})
	//	c.Abort()
	//	return
	//}
	//cqq,_:=strconv.ParseInt(uid,10,64)
	//rsp:=s.bot.CQDeleteMessage(cqq)
	//if rsp["status"]=="ok"{
	//	c.HTML(http.StatusOK, "jump.html", gin.H{
	//		"url":     "/admin/gfriend_list",
	//		"timeout": 3,
	//		"code":    1, //1为success,0为error
	//		"msg":     "删除好友成功",
	//	})
	//	c.Abort()
	//	return
	//}else{
	//	c.HTML(http.StatusOK, "jump.html", gin.H{
	//		"url":     "/admin/friend_list",
	//		"timeout": 3,
	//		"code":   0, //1为success,0为error
	//		"msg":     "删除好友失败",
	//	})
	//	c.Abort()
	//	return
	//}
	c.HTML(http.StatusOK, "admin/jump.html", gin.H{
		"url":     "/admin/friend_list",
		"timeout": 3,
		"code":    0, //1为success,0为error
		"msg":     "功能未实现",
	})
	c.Abort()
	return
}

// 发送好友消息html
func AdminSendPrivateMsg(s *webServer, c *gin.Context) {
	uid, ext := c.GetQuery("uid")
	if !ext {
		c.HTML(http.StatusOK, "admin/jump.html", gin.H{
			"url":     "/admin/friend_list",
			"timeout": "3",
			"code":    0, //1为success,0为error
			"msg":     "缺失参数uid/qq号码",
		})
		c.Abort()
		return
	}
	userId, _ := strconv.ParseInt(uid, 10, 64)
	rsp := s.bot.CQGetVipInfo(userId)
	if rsp["status"] != "ok" {
		c.HTML(http.StatusOK, "admin/jump.html", gin.H{
			"url":     "/admin/friend_list",
			"timeout": 3,
			"code":    0, //1为success,0为error
			"msg":     "获取群信息失败",
		})
		c.Abort()
		return
	}
	Json := gjson.Parse(rsp.ToJson())
	nickname := Json.Get("data.nickname")
	c.HTML(http.StatusOK, "admin/send_private_msg.html", gin.H{
		"userId":   userId,
		"nickname": nickname,
	})
}

// 发送群消息
func AdminDoPrivateMsgSend(s *webServer, c *gin.Context) {
	uid := c.PostForm("uid")
	if uid == "" {
		c.JSON(200, gin.H{
			"code": -1,
			"msg":  "缺失了uid/qq号码",
		})
		c.Abort()
		return
	}
	msg := c.PostForm("msg")
	if msg == "" {
		c.JSON(200, gin.H{
			"code": -1,
			"msg":  "缺失了msg/消息内容",
		})
		c.Abort()
		return
	}
	userId, _ := strconv.ParseInt(uid, 10, 64)
	rsp := s.bot.CQSendPrivateMessage(userId, msg, false)
	c.JSON(200, gin.H{
		"code": 0,
		"msg":  rsp,
	})
}

// 热重启 html
func AdminRestart(s *webServer, c *gin.Context) {
	c.HTML(http.StatusOK, "admin/restart.html", gin.H{})
}

// 热重启
func AdminDoRestart(s *webServer, c *gin.Context) {
	s.DoRelogin()
	c.JSON(200, gin.H{
		"code": 0,
		"msg":  "ok",
	})
	return
}

// 冷重启 html
func AdminRestartDocker(s *webServer, c *gin.Context) {
	c.HTML(http.StatusOK, "admin/restart_docker.html", gin.H{})
}

// 冷重启
func AdminDoRestartDocker(s *webServer, c *gin.Context) {
	Console <- os.Kill
	c.JSON(200, gin.H{
		"code": 0,
		"msg":  "ok",
	})
	return
}

// web输入 html 页面
func AdminWebWrite(s *webServer, c *gin.Context) {
	pic := global.ReadAllText("captcha.jpg")
	var picbase64 string
	if pic != "" {
		input := []byte(pic)
		// base64编码
		picbase64 = base64.StdEncoding.EncodeToString(input)
	}
	c.HTML(http.StatusOK, "admin/web_write.html", gin.H{
		"pic":       pic,
		"picbase64": picbase64,
	})
}

// web输入 处理
func AdminDoWebWrite(s *webServer, c *gin.Context) {
	input := c.PostForm("input")
	WebInput <- input
	//global.WriteAllText("input.txt", input)
	c.JSON(200, gin.H{
		"code": 0,
		"msg":  "ok",
	})
}

func getLogPath() string {
	date := time.Now().Format("2006-01-02")
	dir, _ := os.Getwd()
	sep := string(os.PathSeparator)
	LogsPath := dir + sep + "logs" + sep + string(date) + ".log"
	return LogsPath
}

//读日志
func readLastLine(fname string, size int) (string, error) {
	file, err := os.Open(fname)
	if err != nil {
		return "", err
	}
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		return "", err
	}
	if int64(size) > fi.Size() {
		size = int(fi.Size())
	}
	buf := make([]byte, size)
	n, err := file.ReadAt(buf, fi.Size()-int64(len(buf)))
	if err != nil {
		return "", err
	}
	buf = buf[:n]
	return string(buf), nil
}
