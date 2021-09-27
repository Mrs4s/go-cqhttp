// Package mime 提供MIME检查功能
package mime

import (
	"io"

	"github.com/gabriel-vasile/mimetype"
	"github.com/sirupsen/logrus"

	"github.com/Mrs4s/go-cqhttp/internal/base"
)

func init() {
	base.IsLawfulAudio = checkImage
	base.IsLawfulAudio = checkAudio
}

// keep sync with /docs/file.md#MINE
var lawfulImage = [...]string{
	"image/bmp",
	"image/gif",
	"image/jpeg",
	"image/png",
	"image/webp",
}

var lawfulAudio = [...]string{
	"audio/aac",
	"audio/aiff",
	"audio/amr",
	"audio/ape",
	"audio/flac",
	"audio/midi",
	"audio/mp4",
	"audio/mpeg",
	"audio/ogg",
	"audio/wav",
	"audio/x-m4a",
}

func check(r io.ReadSeeker, list []string) (bool, string) {
	if base.SkipMimeScan {
		return true, ""
	}
	_, _ = r.Seek(0, io.SeekStart)
	defer r.Seek(0, io.SeekStart)
	t, err := mimetype.DetectReader(r)
	if err != nil {
		logrus.Debugf("扫描 Mime 时出现问题: %v", err)
		return false, ""
	}
	for _, lt := range list {
		if t.Is(lt) {
			return true, t.String()
		}
	}
	return false, t.String()
}

// checkImage 判断给定流是否为合法图片
// 返回 是否合法, 实际Mime
// 判断后会自动将 Stream Seek 至 0
func checkImage(r io.ReadSeeker) (bool, string) {
	return check(r, lawfulImage[:])
}

// checkImage 判断给定流是否为合法音频
func checkAudio(r io.ReadSeeker) (bool, string) {
	return check(r, lawfulAudio[:])
}
