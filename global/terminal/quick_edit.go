//go:build !windows

package terminal

// DisableQuickEdit 禁用快速编辑，非Windows系统永远返回nil
func DisableQuickEdit() error {
	return nil
}
