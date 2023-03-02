package terminal

import (
	"fmt"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/Mrs4s/go-cqhttp/internal/base"
)

func setConsoleTitle(title string) error {
	p0, err := syscall.UTF16PtrFromString(title)
	if err != nil {
		return err
	}
	r1, _, err := windows.NewLazySystemDLL("kernel32.dll").NewProc("SetConsoleTitleW").Call(uintptr(unsafe.Pointer(p0)))
	if r1 == 0 {
		return err
	}
	return nil
}

// SetTitle 设置标题为 go-cqhttp `版本` `版权`
func SetTitle() {
	_ = setConsoleTitle(fmt.Sprintf("go-cqhttp "+base.Version+" © 2020 - %d Mrs4s", time.Now().Year()))
}
