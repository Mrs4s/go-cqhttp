// +build !windows,!linux,!darwin

package codec

import "errors"

//EncodeToSilk 将音频编码为Silk
func EncodeToSilk(record []byte, tempName string, useCache bool) ([]byte, error) {
	return nil, errors.New("not supported now")
}
