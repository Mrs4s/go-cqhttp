// Package selfupdate 版本升级检查和自更新
package selfupdate

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/Mrs4s/go-cqhttp/global"
	"github.com/Mrs4s/go-cqhttp/internal/base"
	"github.com/Mrs4s/go-cqhttp/internal/download"
)

func readLine() (str string) {
	console := bufio.NewReader(os.Stdin)
	str, _ = console.ReadString('\n')
	str = strings.TrimSpace(str)
	return
}

func lastVersion() (string, error) {
	r, err := download.Request{URL: "https://api.github.com/repos/Mrs4s/go-cqhttp/releases/latest"}.JSON()
	if err != nil {
		return "", err
	}
	return r.Get("tag_name").Str, nil
}

// CheckUpdate 检查更新
func CheckUpdate() {
	logrus.Infof("正在检查更新.")
	if base.Version == "(devel)" {
		logrus.Warnf("检查更新失败: 使用的 Actions 测试版或自编译版本.")
		return
	}
	latest, err := lastVersion()
	if err != nil {
		logrus.Warnf("检查更新失败: %v", err)
		return
	}
	if global.VersionNameCompare(base.Version, latest) {
		logrus.Infof("当前有更新的 go-cqhttp 可供更新, 请前往 https://github.com/Mrs4s/go-cqhttp/releases 下载.")
		logrus.Infof("当前版本: %v 最新版本: %v", base.Version, latest)
		return
	}
	logrus.Infof("检查更新完成. 当前已运行最新版本.")
}

func binaryName() string {
	goarch := runtime.GOARCH
	if goarch == "arm" {
		goarch += "v7"
	}
	ext := "tar.gz"
	if runtime.GOOS == "windows" {
		ext = "zip"
	}
	return fmt.Sprintf("go-cqhttp_%v_%v.%v", runtime.GOOS, goarch, ext)
}

func checksum(domain, version string) []byte {
	sumURL := fmt.Sprintf("%v/Mrs4s/go-cqhttp/releases/download/%v/go-cqhttp_checksums.txt", domain, version)
	sum, err := download.Request{URL: sumURL}.Bytes()
	if err != nil {
		return nil
	}

	rd := bufio.NewReader(bytes.NewReader(sum))
	for {
		str, err := rd.ReadString('\n')
		if err != nil {
			break
		}
		str = strings.TrimSpace(str)
		if strings.HasSuffix(str, binaryName()) {
			sum, _ := hex.DecodeString(strings.TrimSuffix(str, "  "+binaryName()))
			return sum
		}
	}
	return nil
}

func wait() {
	logrus.Info("按 Enter 继续....")
	readLine()
	os.Exit(0)
}

// SelfUpdate 自更新
func SelfUpdate(domain string) {
	github := "https://github.com"
	if domain == "" {
		domain = github
	}
	logrus.Infof("正在检查更新.")
	latest, err := lastVersion()
	if err != nil {
		logrus.Warnf("获取最新版本失败: %v", err)
		wait()
	}
	url := fmt.Sprintf("%v/Mrs4s/go-cqhttp/releases/download/%v/%v", domain, latest, binaryName())
	if base.Version == latest {
		logrus.Info("当前版本已经是最新版本!")
		wait()
	}
	logrus.Info("当前最新版本为 ", latest)
	logrus.Warn("是否更新(y/N): ")
	r := strings.TrimSpace(readLine())
	if r != "y" && r != "Y" {
		logrus.Warn("已取消更新！")
		wait()
	}
	logrus.Info("正在更新,请稍等...")
	sum := checksum(domain, latest)
	if sum != nil {
		if strings.HasPrefix(url, github) {
			url = "https://ghproxy.com/" + url
		}
		err = update(url, sum)
		if err != nil {
			logrus.Error("更新失败: ", err)
		} else {
			logrus.Info("更新成功!")
		}
	} else {
		logrus.Error("checksum 失败!")
	}
	wait()
}

// writeSumCounter 写入量计算实例
type writeSumCounter struct {
	total uint64
	hash  hash.Hash
}

// Write 方法将写入的byte长度追加至写入的总长度Total中
func (wc *writeSumCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.total += uint64(n)
	wc.hash.Write(p)
	fmt.Printf("\r                                    ")
	fmt.Printf("\rDownloading... %s complete", humanBytes(wc.total))
	return n, nil
}

func logn(n, b float64) float64 {
	return math.Log(n) / math.Log(b)
}

func humanBytes(s uint64) string {
	sizes := []string{"B", "kB", "MB", "GB"} // GB对于go-cqhttp来说已经够用了
	if s < 10 {
		return fmt.Sprintf("%d B", s)
	}
	e := math.Floor(logn(float64(s), 1000))
	suffix := sizes[int(e)]
	val := math.Floor(float64(s)/math.Pow(1000, e)*10+0.5) / 10
	f := "%.0f %s"
	if val < 10 {
		f = "%.1f %s"
	}
	return fmt.Sprintf(f, val, suffix)
}

// FromStream copy form getlantern/go-update
func fromStream(updateWith io.Reader) (err error, errRecover error) {
	updatePath, err := os.Executable()
	updatePath = filepath.Clean(updatePath)
	if err != nil {
		return
	}

	// get the directory the executable exists in
	updateDir := filepath.Dir(updatePath)
	filename := filepath.Base(updatePath)
	// Copy the contents of of newbinary to a the new executable file
	newPath := filepath.Join(updateDir, fmt.Sprintf(".%s.new", filename))
	fp, err := os.OpenFile(newPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return
	}
	// We won't log this error, because it's always going to happen.
	defer func() { _ = fp.Close() }()
	if _, err = bufio.NewReader(updateWith).WriteTo(fp); err != nil {
		logrus.Errorf("Unable to copy data: %v\n", err)
	}

	// if we don't call fp.Close(), windows won't let us move the new executable
	// because the file will still be "in use"
	if err := fp.Close(); err != nil {
		logrus.Errorf("Unable to close file: %v\n", err)
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

	// move the new executable in to become the new program
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
