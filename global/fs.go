package global

import (
	"bufio"
	"bytes"
	"compress/bzip2"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/kardianos/osext"
	"io"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/dustin/go-humanize"
	log "github.com/sirupsen/logrus"
)

var (
	IMAGE_PATH     = path.Join("data", "images")
	IMAGE_PATH_OLD = path.Join("data", "image")
	VOICE_PATH     = path.Join("data", "voices")
	VOICE_PATH_OLD = path.Join("data", "record")
	VIDEO_PATH     = path.Join("data", "videos")
	CACHE_PATH     = path.Join("data", "cache")

	HEADER_AMR  = []byte("#!AMR")
	HEADER_SILK = []byte("\x02#!SILK_V3")

	ErrSyntax = errors.New("syntax error")
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
	data, err = nil, ErrSyntax
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

// UpdateFromStream copy form getlantern/go-update
func UpdateFromStream(updateWith io.Reader) (err error, errRecover error) {
	updatePath, err := osext.Executable()
	if err != nil {
		return
	}
	var newBytes []byte
	// no patch to apply, go on through
	var fileHeader []byte
	bufBytes := bufio.NewReader(updateWith)
	fileHeader, err = bufBytes.Peek(2)
	if err != nil {
		return
	}
	// The content is always bzip2 compressed except when running test, in
	// which case is not prefixed with the magic byte sequence for sure.
	if bytes.Equal([]byte{0x42, 0x5a}, fileHeader) {
		// Identifying bzip2 files.
		updateWith = bzip2.NewReader(bufBytes)
	} else {
		updateWith = io.Reader(bufBytes)
	}
	newBytes, err = ioutil.ReadAll(updateWith)
	if err != nil {
		return
	}
	// get the directory the executable exists in
	updateDir := filepath.Dir(updatePath)
	filename := filepath.Base(updatePath)
	// Copy the contents of of newbinary to a the new executable file
	newPath := filepath.Join(updateDir, fmt.Sprintf(".%s.new", filename))
	fp, err := os.OpenFile(newPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return
	}
	// We won't log this error, because it's always going to happen.
	defer func() { _ = fp.Close() }()
	if _, err = io.Copy(fp, bytes.NewReader(newBytes)); err != nil {
		log.Errorf("Unable to copy data: %v\n", err)
	}

	// if we don't call fp.Close(), windows won't let us move the new executable
	// because the file will still be "in use"
	if err := fp.Close(); err != nil {
		log.Errorf("Unable to close file: %v\n", err)
	}
	// this is where we'll move the executable to so that we can swap in the updated replacement
	oldPath := filepath.Join(updateDir, fmt.Sprintf(".%s.old", filename))

	// delete any existing old exec file - this is necessary on Windows for two reasons:
	// 1. after a successful update, Windows can't remove the .old file because the process is still running
	// 2. windows rename operations fail if the destination file already exists
	_ = os.Remove(oldPath)

	// move the existing executable to a new file in the same directory
	err = os.Rename(updatePath, oldPath)
	if err != nil {
		return
	}

	// move the new exectuable in to become the new program
	err = os.Rename(newPath, updatePath)

	if err != nil {
		// copy unsuccessful
		errRecover = os.Rename(oldPath, updatePath)
	} else {
		// copy successful, remove the old binary
		_ = os.Remove(oldPath)
	}
	return
}