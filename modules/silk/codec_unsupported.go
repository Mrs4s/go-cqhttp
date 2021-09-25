//go:build (!arm && !arm64 && !amd64 && !386) || (!windows && !linux && !darwin) || (windows && arm) || (windows && arm64) || race || nosilk
// +build !arm,!arm64,!amd64,!386 !windows,!linux,!darwin windows,arm windows,arm64 race nosilk

package silk

import "errors"

// encode 将音频编码为Silk
func encode(record []byte, tempName string) ([]byte, error) {
	return nil, errors.New("not supported now")
}

// resample 将silk重新编码为 24000 bit rate
func resample(data []byte) []byte {
	return data
}
