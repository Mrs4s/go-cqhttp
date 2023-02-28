package terminal

import (
	"os"

	"golang.org/x/sys/windows"
)

// EnableVT100 启用颜色、控制字符
func EnableVT100() error {
	stdout := windows.Handle(os.Stdout.Fd())

	var mode uint32
	err := windows.GetConsoleMode(stdout, &mode)
	if err != nil {
		return err
	}

	mode |= windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING // 启用虚拟终端处理
	mode |= windows.ENABLE_PROCESSED_OUTPUT            // 启用处理后的输出

	return windows.SetConsoleMode(stdout, mode)
}
