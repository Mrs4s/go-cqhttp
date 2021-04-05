// Package update 包含go-cqhttp自我更新相关函数
package update

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/kardianos/osext"
	log "github.com/sirupsen/logrus"
)

// WriteCounter 写入量计算实例
type WriteCounter struct {
	Total uint64
}

// Write 方法将写入的byte长度追加至写入的总长度Total中
func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Total += uint64(n)
	wc.PrintProgress()
	return n, nil
}

// PrintProgress 方法将打印当前的总写入量
func (wc *WriteCounter) PrintProgress() {
	fmt.Printf("\r%s", strings.Repeat(" ", 35))
	fmt.Printf("\rDownloading... %s complete", humanize.Bytes(wc.Total))
}

// FromStream copy form getlantern/go-update
func FromStream(updateWith io.Reader) (err error, errRecover error) {
	updatePath, err := osext.Executable()
	if err != nil {
		return
	}
	var newBytes []byte
	// no patch to apply, go on through
	bufBytes := bufio.NewReader(updateWith)
	updateWith = io.Reader(bufBytes)
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
