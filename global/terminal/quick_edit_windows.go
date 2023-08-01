package terminal

import (
	"os"

	"golang.org/x/sys/windows"
)

var inputmode uint32

// RestoreInputMode 还原输入模式
func RestoreInputMode() error {
	if inputmode == 0 {
		return nil
	}
	stdin := windows.Handle(os.Stdin.Fd())
	return windows.SetConsoleMode(stdin, inputmode)
}

// DisableQuickEdit 禁用快速编辑
func DisableQuickEdit() error {
	stdin := windows.Handle(os.Stdin.Fd())

	var mode uint32
	err := windows.GetConsoleMode(stdin, &mode)
	if err != nil {
		return err
	}
	inputmode = mode

	mode &^= windows.ENABLE_QUICK_EDIT_MODE // 禁用快速编辑模式
	mode |= windows.ENABLE_EXTENDED_FLAGS   // 启用扩展标志

	mode &^= windows.ENABLE_MOUSE_INPUT    // 禁用鼠标输入
	mode |= windows.ENABLE_PROCESSED_INPUT // 启用控制输入

	mode &^= windows.ENABLE_INSERT_MODE                           // 禁用插入模式
	mode |= windows.ENABLE_ECHO_INPUT | windows.ENABLE_LINE_INPUT // 启用输入回显&逐行输入

	mode &^= windows.ENABLE_WINDOW_INPUT           // 禁用窗口输入
	mode &^= windows.ENABLE_VIRTUAL_TERMINAL_INPUT // 禁用虚拟终端输入

	return windows.SetConsoleMode(stdin, mode)
}
