//go:build !windows

package terminal

// RunningByDoubleClick 检查是否通过双击直接运行，非Windows系统永远返回false
func RunningByDoubleClick() bool {
	return false
}

// NoMoreDoubleClick 提示用户不要双击运行，非Windows系统永远返回nil
func NoMoreDoubleClick() error {
	return nil
}
