// +build linux windows darwin
// +build 386 amd64 arm arm64

package codec

import (
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"runtime"
)

const (
	silkCachePath = "data/cache"
	encoderPath   = "codec"
)

func downloadCodec(url string) (err error) {
	resp, err := http.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	err = ioutil.WriteFile(getEncoderFilePath(), body, os.ModePerm)
	return
}

func getEncoderFilePath() string {
	encoderFile := path.Join(encoderPath, runtime.GOOS+"-"+runtime.GOARCH+"-encoder")
	if runtime.GOOS == "windows" {
		encoderFile = encoderFile + ".exe"
	}
	return encoderFile
}

//Init 下载Silk编码器
func Init() error {
	if !fileExist(silkCachePath) {
		_ = os.MkdirAll(silkCachePath, os.ModePerm)
	}
	if !fileExist(encoderPath) {
		_ = os.MkdirAll(encoderPath, os.ModePerm)
	}
	p := getEncoderFilePath()
	if !fileExist(p) {
		if err := downloadCodec("https://cdn.jsdelivr.net/gh/wdvxdr1123/tosilk/codec/" + runtime.GOOS + "-" + runtime.GOARCH + "-encoder"); err != nil {
			return errors.New("下载依赖失败")
		}
	}
	return nil
}

//EncodeToSilk 将音频编码为Silk
func EncodeToSilk(record []byte, tempName string, useCache bool) ([]byte, error) {
	// 1. 写入缓存文件
	rawPath := path.Join(silkCachePath, tempName+".wav")
	err := ioutil.WriteFile(rawPath, record, os.ModePerm)
	if err != nil {
		return nil, errors.Wrap(err, "write temp file error")
	}
	defer os.Remove(rawPath)

	// 2.转换pcm
	pcmPath := path.Join(silkCachePath, tempName+".pcm")
	cmd := exec.Command("ffmpeg", "-i", rawPath, "-f", "s16le", "-ar", "24000", "-ac", "1", pcmPath)
	if err = cmd.Run(); err != nil {
		return nil, errors.Wrap(err, "convert pcm file error")
	}
	defer os.Remove(pcmPath)

	// 3. 转silk
	silkPath := path.Join(silkCachePath, tempName+".silk")
	cmd = exec.Command(getEncoderFilePath(), pcmPath, silkPath, "-rate", "24000", "-quiet", "-tencent")
	if err = cmd.Run(); err != nil {
		return nil, errors.Wrap(err, "convert silk file error")
	}
	if !useCache {
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
