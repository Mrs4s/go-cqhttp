// Package mime 提供MIME检查功能
package mime

import (
	"io"
	"net/http"
	"strings"

	"github.com/Mrs4s/go-cqhttp/internal/base"
)

func init() {
	base.IsLawfulImage = checkImage
	base.IsLawfulAudio = checkAudio
}

const limit = 4 * 1024

func scan(r io.ReadSeeker) string {
	_, _ = r.Seek(0, io.SeekStart)
	defer r.Seek(0, io.SeekStart)
	in := make([]byte, limit)
	_, _ = r.Read(in)
	return http.DetectContentType(in)
}

// checkImage 判断给定流是否为合法图片
// 返回 是否合法, 实际Mime
// 判断后会自动将 Stream Seek 至 0
func checkImage(r io.ReadSeeker) (ok bool, t string) {
	if base.SkipMimeScan {
		return true, ""
	}
	t = scan(r)
	switch t {
	case "image/bmp", "image/gif", "image/jpeg", "image/png", "image/webp":
		ok = true
	}
	return
}

// checkImage 判断给定流是否为合法音频
func checkAudio(r io.ReadSeeker) (bool, string) {
	if base.SkipMimeScan {
		return true, ""
	}
	t := scan(r)
	// std mime type detection is not full supported for audio
	if strings.Contains(t, "text") || strings.Contains(t, "image") {
		return false, t
	}
	return true, t
}
