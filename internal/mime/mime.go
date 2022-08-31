// Package mime 提供MIME检查功能
package mime

import (
	"io"
	"net/http"
	"strings"

	"github.com/Mrs4s/go-cqhttp/internal/base"
)

const limit = 4 * 1024

func scan(r io.ReadSeeker) string {
	_, _ = r.Seek(0, io.SeekStart)
	defer r.Seek(0, io.SeekStart)
	in := make([]byte, limit)
	_, _ = r.Read(in)
	return http.DetectContentType(in)
}

// CheckImage 判断给定流是否为合法图片
// 返回 是否合法, 实际Mime
// 判断后会自动将 Stream Seek 至 0
func CheckImage(r io.ReadSeeker) (t string, ok bool) {
	if base.SkipMimeScan {
		return "", true
	}
	if r == nil {
		return "image/nil-stream", false
	}
	t = scan(r)
	switch t {
	case "image/bmp", "image/gif", "image/jpeg", "image/png", "image/webp":
		ok = true
	}
	return
}

// CheckAudio 判断给定流是否为合法音频
func CheckAudio(r io.ReadSeeker) (string, bool) {
	if base.SkipMimeScan {
		return "", true
	}
	t := scan(r)
	// std mime type detection is not full supported for audio
	if strings.Contains(t, "text") || strings.Contains(t, "image") {
		return t, false
	}
	return t, true
}
