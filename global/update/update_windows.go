package update

import (
	"archive/zip"
	"bytes"
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
	reader, _ := zip.NewReader(bytes.NewReader(rsp), resp.ContentLength)
	file, err := reader.Open("go-cqhttp.exe")
	if err != nil {
		return err
	}
	err, _ = FromStream(file)
	fmt.Println()
	if err != nil {
		return err
	}
	return nil
}
