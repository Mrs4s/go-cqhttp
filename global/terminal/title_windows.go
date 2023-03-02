package terminal

import (
	"fmt"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/Mrs4s/go-cqhttp/internal/base"
)

var (
	//go:linkname modkernel32 golang.org/x/sys/windows.modkernel32
	modkernel32         *windows.LazyDLL
	procSetConsoleTitle = modkernel32.NewProc("SetConsoleTitleW")
)

//go:linkname errnoErr golang.org/x/sys/windows.errnoErr
func errnoErr(e syscall.Errno) error

func setConsoleTitle(title string) (err error) {
	var p0 *uint16
	p0, err = syscall.UTF16PtrFromString(title)
	if err != nil {
		return
	}
	r1, _, e1 := syscall.Syscall(procSetConsoleTitle.Addr(), 1, uintptr(unsafe.Pointer(p0)), 0, 0)
	if r1 == 0 {
		err = errnoErr(e1)
	}
	return
}

func init() {
	_ = setConsoleTitle(fmt.Sprintf("go-cqhttp "+base.Version+" © 2020 - %d Mrs4s", time.Now().Year()))
}
