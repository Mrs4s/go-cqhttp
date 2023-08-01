//go:build !windows

package terminal

// EnableVT100 启用颜色、控制字符，非Windows系统永远返回nil
func EnableVT100() error {
	return nil
}
