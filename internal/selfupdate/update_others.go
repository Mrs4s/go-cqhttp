//go:build !windows
// +build !windows

package selfupdate

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"errors"
	"io"
	"net/http"

	"github.com/klauspost/compress/gzip"
)

// update go-cqhttp自我更新
func update(url string, sum []byte) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	wc := writeSumCounter{
		hash: sha256.New(),
	}
	rsp, err := io.ReadAll(io.TeeReader(resp.Body, &wc))
	if err != nil {
		return err
	}
	if !bytes.Equal(wc.hash.Sum(nil), sum) {
		return errors.New("文件已损坏")
	}
	gr, err := gzip.NewReader(bytes.NewReader(rsp))
	if err != nil {
		return err
	}
	tr := tar.NewReader(gr)
	for {
		header, err := tr.Next()
		if err != nil {
			return err
		}
		if header.Name == "go-cqhttp" {
			err, _ := fromStream(tr)
			if err != nil {
				return err
			}
			return nil
		}
	}
}
