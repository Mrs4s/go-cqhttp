// +build windows

package terminal

import (
	"syscall"
	"unsafe"
)

// RunningByDoubleClick 检查是否通过双击直接运行
func RunningByDoubleClick() bool {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	lp := kernel32.NewProc("GetConsoleProcessList")
	if lp != nil {
		var ids [2]uint32
		var maxCount uint32 = 2
		ret, _, _ := lp.Call(uintptr(unsafe.Pointer(&ids)), uintptr(maxCount))
		if ret > 1 {
			return false
		}
	}
	return true
}
