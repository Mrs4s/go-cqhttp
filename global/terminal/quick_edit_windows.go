package terminal

import "golang.org/x/sys/windows"

// DisableQuickEdit 禁用快速编辑
func DisableQuickEdit() error {
	stdin := windows.Handle(os.Stdin.Fd())

	var mode uint32
	err := windows.GetConsoleMode(stdin, &mode)
	if err != nil {
		return err
	}

	mode &^= windows.ENABLE_QUICK_EDIT_MODE // 禁用快速编辑模式
	mode |= windows.ENABLE_EXTENDED_FLAGS   // 启用扩展标志

	return windows.SetConsoleMode(stdin, mode)
}
