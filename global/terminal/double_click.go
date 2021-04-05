// +build !windows

package terminal

// RunningByDoubleClick 检查是否通过双击直接运行,非Windows系统永远返回false
func RunningByDoubleClick() bool {
	return false
}
