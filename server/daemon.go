//daemon 功能写在这，目前仅支持了-d 作为后台运行参数，stop，start，restart这些功能目前看起来并不需要，可以通过api控制，后续需要的话再补全。
package server

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

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

	proc := exec.Command(os.Args[0], execArgs...)
	err := proc.Start()

	if err != nil {
		panic(err)
	}

	fmt.Println("[PID] ", proc.Process.Pid)

	os.Exit(0)
}

func GetCurrentPath() (string, error) {
	file, err := exec.LookPath(os.Args[0])
	if err != nil {
		return "", err
	}
	path, err := filepath.Abs(file)
	if err != nil {
		return "", err
	}
	//fmt.Println("path111:", path)
	if runtime.GOOS == "windows" {
		path = strings.Replace(path, "\\", "/", -1)
	}
	//fmt.Println("path222:", path)
	i := strings.LastIndex(path, "/")
	if i < 0 {
		//return "", errors.New("system/path_error", `Can't find "/" or "\".`)
	}
	//fmt.Println("path333:", path)
	return string(path[0 : i+1]), nil
}
