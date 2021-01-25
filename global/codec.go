package global

import (
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"os/exec"
	"path"

	"github.com/Mrs4s/go-cqhttp/global/codec"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

var useSilkCodec = true

//InitCodec 初始化Silk编码器
func InitCodec() {
	log.Info("正在加载silk编码器...")
	err := codec.Init()
	if err != nil {
		log.Error(err)
		useSilkCodec = false
	}
}

//EncoderSilk 将音频编码为Silk
func EncoderSilk(data []byte) ([]byte, error) {
	if !useSilkCodec {
		return nil, errors.New("no silk encoder")
	}
	h := md5.New()
	_, err := h.Write(data)
	if err != nil {
		return nil, errors.Wrap(err, "calc md5 failed")
	}
	tempName := fmt.Sprintf("%x", h.Sum(nil))
	if silkPath := path.Join("data/cache", tempName+".silk"); PathExists(silkPath) {
		return ioutil.ReadFile(silkPath)
	}
	slk, err := codec.EncodeToSilk(data, tempName, true)
	if err != nil {
		return nil, errors.Wrap(err, "encode silk failed")
	}
	return slk, nil
}

//EncodeMP4 将给定视频文件编码为MP4
func EncodeMP4(src string, dst string) error { //        -y 覆盖文件
	cmd1 := exec.Command("ffmpeg", "-i", src, "-y", "-c", "copy", "-map", "0", dst)
	err := cmd1.Run()
	if err != nil {
		cmd2 := exec.Command("ffmpeg", "-i", src, "-y", "-c:v", "h264", "-c:a", "mp3", dst)
		return errors.Wrap(cmd2.Run(), "convert mp4 failed")
	}
	return err
}

//ExtractCover 获取给定视频文件的Cover
func ExtractCover(src string, target string) error {
	cmd := exec.Command("ffmpeg", "-i", src, "-y", "-r", "1", "-f", "image2", target)
	return errors.Wrap(cmd.Run(), "extract video cover failed")
}
