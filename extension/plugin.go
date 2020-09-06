package extension

import (
	"fmt"
	"github.com/Mrs4s/go-cqhttp/coolq"
	"github.com/Mrs4s/go-cqhttp/extension/modules/client"
	messageModule "github.com/Mrs4s/go-cqhttp/extension/modules/message"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
)

type Plugin *goja.Runtime

type Pool struct {
	pool []Plugin
	bot *coolq.CQBot
}

var pluginPool = Pool{}

func Run(bot *coolq.CQBot) {
	pluginPool.bot = bot
	plg := goja.New()
	plg.Set("CQBot", pluginPool.bot)
	plg.Set("CQClient", pluginPool.bot.Client)
	registry := new(require.Registry)
	registry.Enable(plg)
	clientModule.Enable(plg)
	messageModule.Enable(plg)
	jsFile, err := ioutil.ReadFile("test.js")
	if err != nil {
		fmt.Println(err)
	}
	_, err = plg.RunString(string(jsFile))
	_, err = plg.RunString(`on_create()`)
	if err != nil {
		fmt.Println(err)
	}
	pluginPool.pool = append(pluginPool.pool, plg)
}

func scanDir(dirName string) []string {
	files, err := ioutil.ReadDir(dirName)
	if err != nil {
		log.Println(err)
	}
	var fileList []string
	for _, file := range files {
		fileList = append(fileList, dirName + string(os.PathSeparator) + file.Name())
	}
	return fileList
}