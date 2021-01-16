package global

import (
	"crypto/md5"
	"errors"
	"fmt"
	"github.com/Mrs4s/go-cqhttp/global/codec"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os/exec"
	"path"
)

var useSilkCodec = true

func InitCodec() {
	log.Info("正在加载silk编码器...")
	err := codec.Init()
	if err != nil {
		log.Error(err)
		useSilkCodec = false
	}
}

func EncoderSilk(data []byte) ([]byte, error) {
	if !useSilkCodec {
		return nil, errors.New("no silk encoder")
	}
	h := md5.New()
	_, err := h.Write(data)
	if err != nil {
		return nil, err
	}
	tempName := fmt.Sprintf("%x", h.Sum(nil))
	if silkPath := path.Join("data/cache", tempName+".silk"); PathExists(silkPath) {
		return ioutil.ReadFile(silkPath)
	}
	slk, err := codec.EncodeToSilk(data, tempName, true)
	if err != nil {
		return nil, err
	}
	return slk, nil
}

func EncodeMP4(src string, dst string) error { //        -y 覆盖文件
	cmd := exec.Command("ffmpeg", "-i", src, "-y", "-c", "copy", "-map", "0", dst)
	return cmd.Run()
}

func ExtractCover(src string, dst string) error {
	cmd := exec.Command("ffmpeg", "-i", src, "-y", "-r", "1", "-f", "image2", dst)
	return cmd.Run()
}
