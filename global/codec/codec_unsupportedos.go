// +build !windows,!linux,!darwin

package codec

import "errors"

func Init() error {
	return errors.New("not support now")
}

func EncodeToSilk(record []byte, tempName string, useCache bool) ([]byte, error) {
	return nil, errors.New("not support now")
}
