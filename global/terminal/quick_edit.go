//go:build !windows

package terminal

// RestoreInputMode 还原输入模式，非Windows系统永远返回nil
func RestoreInputMode() error {
	return nil
}

// DisableQuickEdit 禁用快速编辑，非Windows系统永远返回nil
func DisableQuickEdit() error {
	return nil
}
