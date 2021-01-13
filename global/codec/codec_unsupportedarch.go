// +build !386,!arm64,!amd64,!arm

package codec

import "errors"

func Init(cachePath, codecPath string) error {
	return errors.New("Unsupport arch now")
}

func EncodeToSilk(record []byte, tempName string, useCache bool) ([]byte, error) {
	return nil, errors.New("Unsupport arch now")
}
