//go:build (linux || (windows && !arm && !arm64) || darwin) && (386 || amd64 || arm || arm64) && !race && !nosilk
// +build linux windows,!arm,!arm64 darwin
// +build 386 amd64 arm arm64
// +build !race
// +build !nosilk

package silk

import (
	"os"
	"os/exec"
	"path"

	"github.com/pkg/errors"
	"github.com/wdvxdr1123/go-silk"

	"github.com/Mrs4s/go-cqhttp/internal/base"
)

const silkCachePath = "data/cache"

// encode 将音频编码为Silk
func encode(record []byte, tempName string) (silkWav []byte, err error) {
	// 1. 写入缓存文件
	rawPath := path.Join(silkCachePath, tempName+".wav")
	err = os.WriteFile(rawPath, record, os.ModePerm)
	if err != nil {
		return nil, errors.Wrap(err, "write temp file error")
	}
	defer os.Remove(rawPath)

	// 2.转换pcm
	pcmPath := path.Join(silkCachePath, tempName+".pcm")
	cmd := exec.Command("ffmpeg", "-i", rawPath, "-f", "s16le", "-ar", "24000", "-ac", "1", pcmPath)
	if errors.Is(cmd.Err, exec.ErrDot) {
		cmd.Err = nil
	}
	if base.Debug {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	if err = cmd.Run(); err != nil {
		return nil, errors.Wrap(err, "convert pcm file error")
	}
	defer os.Remove(pcmPath)

	// 3. 转silk
	pcm, err := os.ReadFile(pcmPath)
	if err != nil {
		return nil, errors.Wrap(err, "read pcm file err")
	}
	silkWav, err = silk.EncodePcmBuffToSilk(pcm, 24000, 24000, true)
	if err != nil {
		return nil, errors.Wrap(err, "silk encode error")
	}
	silkPath := path.Join(silkCachePath, tempName+".silk")
	err = os.WriteFile(silkPath, silkWav, 0o666)
	return
}

// resample 将silk重新编码为 24000 bit rate
func resample(data []byte) []byte {
	pcm, err := silk.DecodeSilkBuffToPcm(data, 24000)
	if err != nil {
		panic(err)
	}
	data, err = silk.EncodePcmBuffToSilk(pcm, 24000, 24000, true)
	if err != nil {
		panic(err)
	}
	return data
}
