package extension

import (
	"fmt"
	"github.com/Mrs4s/MiraiGo/client"
	"github.com/Mrs4s/MiraiGo/message"
	"github.com/Mrs4s/go-cqhttp/extension/modules/client"
	messageModule "github.com/Mrs4s/go-cqhttp/extension/modules/message"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"io/ioutil"
	"os"
)

type Plugins struct {
	pool   []*goja.Runtime
	client *client.QQClient
}

var plugins = Plugins{}
var pluginEnable = false

func Run(client *client.QQClient) {
	plugins.client = client
	pluginEnable = true
	filelist := scanDir("plugin")
	for _, file := range filelist {
		log.Infof("正在加载插件 %v", file)
		plg := goja.New()
		plg.Set("CQClient", plugins.client)
		registry := new(require.Registry)
		registry.Enable(plg)
		clientModule.Enable(plg)
		messageModule.Enable(plg)
		jsFile, err := ioutil.ReadFile(file)
		if err != nil {
			fmt.Println(err)
		}
		_, err = plg.RunString(string(jsFile))
		if err != nil {
			fmt.Println(err)
		}
		_, err = plg.RunString(`on_create()`)
		if err != nil {
			fmt.Println(err)
		}
		plugins.pool = append(plugins.pool, plg)
	}
}

func scanDir(dirName string) []string {
	files, err := ioutil.ReadDir(dirName)
	if err != nil {
		log.Println(err)
	}
	var fileList []string
	for _, file := range files {
		if file.IsDir() == false {
			fileList = append(fileList, dirName+string(os.PathSeparator)+file.Name())
		}
	}
	return fileList
}

func OnMissedAction(action string, params map[string]string) (r map[string]interface{}) {
	if pluginEnable == false {
		return nil
	}
	for _, plg := range plugins.pool {
		var fn func(string, map[string]string) string
		err := plg.ExportTo(plg.Get("on_missed_action"), &fn)
		if err != nil {
			log.Warn(err)
		} else {
			ret := fn(action, params)
			if ret != "" {
				r = make(map[string]interface{})
				gjson.Parse(ret).ForEach(func(key, value gjson.Result) bool {
					r[key.Str] = value.Str
					return true
				})
			}
		}
	}
	return
}

func OnMissedCQCode(code string, params map[string]string) message.IMessageElement {
	if pluginEnable == false {
		return nil
	}
	for _, plg := range plugins.pool {
		var fn func(string, map[string]string) message.IMessageElement
		err := plg.ExportTo(plg.Get("on_missed_cqcode"), &fn)
		if err != nil {
			log.Warn(err)
		} else {
			r := fn(code, params)
			if r != nil {
				return r
			}
		}
	}
	return nil
}
