// +build linux windows darwin
// +build 386 amd64 arm

package codec

import (
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"runtime"
)

var (
	codecDir    string
	encoderPath string
	cachePath   string
)

func downloadCodec(url string, path string) (err error) {
	resp, err := http.Get(url)
	if runtime.GOOS == "windows" {
		path = path + ".exe"
	}
	if err != nil {
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	err = ioutil.WriteFile(path, body, os.ModePerm)
	return
}

func Init(cachePath, codecPath string) error {
	appPath, err := os.Executable()
	appPath = path.Dir(appPath)
	if err != nil {
		return err
	}
	cachePath = path.Join(appPath, cachePath)
	codecDir = path.Join(appPath, codecPath)
	if !fileExist(codecDir) {
		_ = os.MkdirAll(codecDir, os.ModePerm)
	}
	if !fileExist(cachePath) {
		_ = os.MkdirAll(cachePath, os.ModePerm)
	}
	encoderFile := runtime.GOOS + "-" + runtime.GOARCH + "-encoder"
	encoderPath = path.Join(codecDir, encoderFile)
	if !fileExist(encoderPath) {
		if err = downloadCodec("https://cdn.jsdelivr.net/gh/wdvxdr1123/tosilk/codec/"+encoderFile, encoderPath); err != nil {
			return errors.New("下载依赖失败")
		}
	}
	if runtime.GOOS == "windows" {
		encoderPath = encoderPath + ".exe"
	}
	return nil
}

func EncodeToSilk(record []byte, tempName string, useCache bool) ([]byte, error) {
	// 1. 写入缓存文件
	rawPath := path.Join(cachePath, tempName+".wav")
	err := ioutil.WriteFile(rawPath, record, os.ModePerm)
	if err != nil {
		return nil, err
	}
	defer os.Remove(rawPath)

	// 2.转换pcm
	pcmPath := path.Join(cachePath, tempName+".pcm")
	cmd := exec.Command("ffmpeg", "-i", rawPath, "-f", "s16le", "-ar", "24000", "-ac", "1", pcmPath)
	if err = cmd.Run(); err != nil {
		return nil, err
	}
	defer os.Remove(pcmPath)

	// 3. 转silk
	silkPath := path.Join(cachePath, tempName+".silk")
	cmd = exec.Command(encoderPath, pcmPath, silkPath, "-rate", "24000", "-quiet", "-tencent")
	if err = cmd.Run(); err != nil {
		return nil, err
	}
	if useCache == false {
		defer os.Remove(silkPath)
	}
	return ioutil.ReadFile(silkPath)
}

// FileExist 检查文件是否存在
func fileExist(path string) bool {
	if runtime.GOOS == "windows" {
		path = path + ".exe"
	}
	_, err := os.Lstat(path)
	return !os.IsNotExist(err)
}
