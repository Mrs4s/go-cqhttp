package global

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"

	"github.com/dustin/go-humanize"
	log "github.com/sirupsen/logrus"
)

var (
	IMAGE_PATH = path.Join("data", "images")
	VOICE_PATH = path.Join("data", "voices")
	VIDEO_PATH = path.Join("data", "videos")
	CACHE_PATH = path.Join("data", "cache")

	HEADER_AMR  = []byte("#!AMR")
	HEADER_SILK = []byte("\x02#!SILK_V3")
)

func PathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

func ReadAllText(path string) string {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(b)
}

func WriteAllText(path, text string) error {
	return ioutil.WriteFile(path, []byte(text), 0644)
}

func Check(err error) {
	if err != nil {
		log.Fatalf("遇到错误: %v", err)
	}
}

func IsAMRorSILK(b []byte) bool {
	return bytes.HasPrefix(b, HEADER_AMR) || bytes.HasPrefix(b, HEADER_SILK)
}

func FindFile(f, cache, PATH string) (data []byte, err error) {
	data, err = nil, errors.New("syntax error")
	if strings.HasPrefix(f, "http") || strings.HasPrefix(f, "https") {
		if cache == "" {
			cache = "1"
		}
		hash := md5.Sum([]byte(f))
		cacheFile := path.Join(CACHE_PATH, hex.EncodeToString(hash[:])+".cache")
		if PathExists(cacheFile) && cache == "1" {
			return ioutil.ReadFile(cacheFile)
		}
		data, err = GetBytes(f)
		_ = ioutil.WriteFile(cacheFile, data, 0644)
		if err != nil {
			return nil, err
		}
	} else if strings.HasPrefix(f, "base64") {
		data, err = base64.StdEncoding.DecodeString(strings.ReplaceAll(f, "base64://", ""))
		if err != nil {
			return nil, err
		}
	} else if strings.HasPrefix(f, "file") {
		var fu *url.URL
		fu, err = url.Parse(f)
		if err != nil {
			return nil, err
		}
		if strings.HasPrefix(fu.Path, "/") && runtime.GOOS == `windows` {
			fu.Path = fu.Path[1:]
		}
		data, err = ioutil.ReadFile(fu.Path)
		if err != nil {
			return nil, err
		}
	} else if PathExists(path.Join(PATH, f)) {
		data, err = ioutil.ReadFile(path.Join(PATH, f))
		if err != nil {
			return nil, err
		}
	}
	return
}

func DelFile(path string) bool {
	err := os.Remove(path)
	if err != nil {
		// 删除失败
		log.Error(err)
		return false
	} else {
		// 删除成功
		log.Info(path + "删除成功")
		return true
	}
}

func ReadAddrFile(path string) []*net.TCPAddr {
	d, err := ioutil.ReadFile(path)
	if err != nil {
		return nil
	}
	str := string(d)
	lines := strings.Split(str, "\n")
	var ret []*net.TCPAddr
	for _, l := range lines {
		ip := strings.Split(strings.TrimSpace(l), ":")
		if len(ip) == 2 {
			port, _ := strconv.Atoi(ip[1])
			ret = append(ret, &net.TCPAddr{IP: net.ParseIP(ip[0]), Port: port})
		}
	}
	return ret
}

type WriteCounter struct {
	Total uint64
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Total += uint64(n)
	wc.PrintProgress()
	return n, nil
}

func (wc WriteCounter) PrintProgress() {
	fmt.Printf("\r%s", strings.Repeat(" ", 35))
	fmt.Printf("\rDownloading... %s complete", humanize.Bytes(wc.Total))
}
