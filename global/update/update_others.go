// +build !windows

package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// Update go-cqhttp自我更新
func Update(url string, sum []byte) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	wc := WriteSumCounter{
		Hash: sha256.New(),
	}
	rsp, err := io.ReadAll(io.TeeReader(resp.Body, &wc))
	if err != nil {
		return err
	}
	if !bytes.Equal(wc.Hash.Sum(nil), sum) {
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
			err, _ := FromStream(tr)
			fmt.Println()
			if err != nil {
				return err
			}
			return nil
		}
	}
}
