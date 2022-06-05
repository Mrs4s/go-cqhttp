package global

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"net/netip"
	"net/url"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/Mrs4s/MiraiGo/utils"
	log "github.com/sirupsen/logrus"

	"github.com/Mrs4s/go-cqhttp/internal/param"
)

const (
	// ImagePath go-cqhttp使用的图片缓存目录
	ImagePath = "data/images"
	// VoicePath go-cqhttp使用的语音缓存目录
	VoicePath = "data/voices"
	// VideoPath go-cqhttp使用的视频缓存目录
	VideoPath = "data/videos"
	// CachePath go-cqhttp使用的缓存目录
	CachePath = "data/cache"
	// DumpsPath go-cqhttp使用错误转储目录
	DumpsPath = "dumps"
	// HeaderAmr AMR文件头
	HeaderAmr = "#!AMR"
	// HeaderSilk Silkv3文件头
	HeaderSilk = "\x02#!SILK_V3"
)

// PathExists 判断给定path是否存在
func PathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil || errors.Is(err, os.ErrExist)
}

// ReadAllText 读取给定path对应文件，无法读取时返回空值
func ReadAllText(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		log.Error(err)
		return ""
	}
	return string(b)
}

// WriteAllText 将给定text写入给定path
func WriteAllText(path, text string) error {
	return os.WriteFile(path, utils.S2B(text), 0o644)
}

// Check 检测err是否为nil
func Check(err error, deleteSession bool) {
	if err != nil {
		if deleteSession && PathExists("session.token") {
			_ = os.Remove("session.token")
		}
		log.Fatalf("遇到错误: %v", err)
	}
}

// IsAMRorSILK 判断给定文件是否为Amr或Silk格式
func IsAMRorSILK(b []byte) bool {
	return bytes.HasPrefix(b, []byte(HeaderAmr)) || bytes.HasPrefix(b, []byte(HeaderSilk))
}

// FindFile 从给定的File寻找文件，并返回文件byte数组。File是一个合法的URL。p为文件寻找位置。
// 对于HTTP/HTTPS形式的URL，Cache为"1"或空时表示启用缓存
func FindFile(file, cache, p string) (data []byte, err error) {
	data, err = nil, os.ErrNotExist
	switch {
	case strings.HasPrefix(file, "http"): // https also has prefix http
		hash := md5.Sum([]byte(file))
		cacheFile := path.Join(CachePath, hex.EncodeToString(hash[:])+".cache")
		if (cache == "" || cache == "1") && PathExists(cacheFile) {
			return os.ReadFile(cacheFile)
		}
		data, err = GetBytes(file)
		_ = os.WriteFile(cacheFile, data, 0o644)
		if err != nil {
			return nil, err
		}
	case strings.HasPrefix(file, "base64"):
		data, err = param.Base64DecodeString(strings.TrimPrefix(file, "base64://"))
		if err != nil {
			return nil, err
		}
	case strings.HasPrefix(file, "file"):
		var fu *url.URL
		fu, err = url.Parse(file)
		if err != nil {
			return nil, err
		}
		if strings.HasPrefix(fu.Path, "/") && runtime.GOOS == `windows` {
			fu.Path = fu.Path[1:]
		}
		data, err = os.ReadFile(fu.Path)
		if err != nil {
			return nil, err
		}
	case PathExists(path.Join(p, file)):
		data, err = os.ReadFile(path.Join(p, file))
		if err != nil {
			return nil, err
		}
	}
	return
}

// DelFile 删除一个给定path，并返回删除结果
func DelFile(path string) bool {
	err := os.Remove(path)
	if err != nil {
		// 删除失败
		log.Error(err)
		return false
	}
	// 删除成功
	log.Info(path + "删除成功")
	return true
}

// ReadAddrFile 从给定path中读取合法的IP地址与端口,每个IP地址以换行符"\n"作为分隔
func ReadAddrFile(path string) []netip.AddrPort {
	d, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	str := string(d)
	lines := strings.Split(str, "\n")
	var ret []netip.AddrPort
	for _, l := range lines {
		addr, err := netip.ParseAddrPort(l)
		if err == nil {
			ret = append(ret, addr)
		}
	}
	return ret
}
