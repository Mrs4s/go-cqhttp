package server

// daemon 功能写在这，目前仅支持了-d 作为后台运行参数，stop，start，restart这些功能目前看起来并不需要，可以通过api控制，后续需要的话再补全。

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Mrs4s/go-cqhttp/global"

	log "github.com/sirupsen/logrus"
)

// Daemon go-cqhttp server 的 daemon的实现函数
func Daemon() {
	args := os.Args[1:]

	execArgs := make([]string, 0)

	l := len(args)
	for i := 0; i < l; i++ {
		if strings.Index(args[i], "-d") == 0 {
			continue
		}

		execArgs = append(execArgs, args[i])
	}

	ex, _ := os.Executable()
	p, _ := filepath.Abs(ex)
	proc := exec.Command(p, execArgs...)
	err := proc.Start()
	if err != nil {
		panic(err)
	}

	log.Info("[PID] ", proc.Process.Pid)
	// pid写入到pid文件中，方便后续stop的时候kill
	pidErr := savePid("go-cqhttp.pid", strconv.FormatInt(int64(proc.Process.Pid), 10))
	if pidErr != nil {
		log.Errorf("save pid file error: %v", pidErr)
	}

	os.Exit(0)
}

// savePid 保存pid到文件中，便于后续restart/stop的时候kill pid用。
func savePid(path string, data string) error {
	return global.WriteAllText(path, data)
}
