// +build !windows,!linux,!darwin

package codec

import "errors"

//Init 下载silk编码器
func Init() error {
	return errors.New("not support now")
}

//EncodeToSilk 将音频编码为Silk
func EncodeToSilk(record []byte, tempName string, useCache bool) ([]byte, error) {
	return nil, errors.New("not support now")
}
