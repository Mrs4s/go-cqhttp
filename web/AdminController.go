package web

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"html/template"
	"net/http"
	"os"
	"strconv"
	"time"
)

// admin 子站的 路由映射
var HttpuriAdmin = map[string]func(s *webServer, c *gin.Context){
	"index":             AdminIndex,
	"config_json":       AdminConfigJson,
	"config":            AdminConfig,
	"debug":             AdminDebug,
	"log":               AdminLog,
	"friend_list":       AdminFriendList,
	"group_list":        AdminGroupList,
	"get_log":           AdmingetLogs,
	"do_friend_list":    AdminDoFriendList,
	"do_group_list":     AdminDoGroupList,
	"do_config_base":    AdminDoConfigBase,
	"do_config_http":    AdminDoConfigHttp,
	"do_config_ws":      AdminDoConfigWs,
	"do_config_reverse": AdminDoConfigReverse,
	"do_config_json":    AdminDoConfigJson,
}

// 首页
func AdminIndex(s *webServer, c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{})
}

//json config
func AdminConfigJson(s *webServer, c *gin.Context) {
	conf := GetConf()
	data, _ := json.MarshalIndent(conf, "", "\t")
	c.HTML(http.StatusOK, "config_json.html", gin.H{
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
	c.HTML(http.StatusOK, "config.html", gin.H{
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
	c.HTML(http.StatusOK, "log.html", gin.H{
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
	c.HTML(http.StatusOK, "jump.html", gin.H{
		"url":     "/admin/index",
		"timeout": "3",
		"code":    0, //1为success,0为error
		"msg":     "开发中",
	})
	//c.JSON(http.StatusOK, gin.H{"code": -1, "msg": "开发中"})
}

// 好友列表html页面
func AdminFriendList(s *webServer, c *gin.Context) {
	c.HTML(http.StatusOK, "friend_list.html", gin.H{})
}

// 群列表html页面
func AdminGroupList(s *webServer, c *gin.Context) {
	c.HTML(http.StatusOK, "group_list.html", gin.H{})
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

func getLogPath() string {
	date := time.Now().Format("2006-01-02")
	dir, _ := os.Getwd()
	sep := string(os.PathSeparator)
	LogsPath := dir + sep + "logs" + sep + string(date) + ".log"
	//conf:=GetConf()
	//logLevel := conf.LogLevel
	//if logLevel != ""{
	//	switch conf.LogLevel {
	//	case "warn":
	//		LogsPath= dir + sep + "logs" + sep + string(date) + "-"+logLevel+".log"
	//	case "error":
	//		LogsPath= dir + sep + "logs" + sep + string(date) + "-"+logLevel+".log"
	//	default:
	//		LogsPath= dir + sep + "logs" + sep + string(date) + "-warn"+".log"
	//	}
	//}
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
	buf := make([]byte, size)
	n, err := file.ReadAt(buf, fi.Size()-int64(len(buf)))
	if err != nil {
		return "", err
	}
	buf = buf[:n]
	return string(buf), nil
}
