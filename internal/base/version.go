package base

import "runtime/debug"

// Version go-cqhttp的版本信息，在编译时使用ldflags进行覆盖
var Version = "unknown"

func init() {
	if Version != "unknown" {
		return
	}
	info, ok := debug.ReadBuildInfo()
	if ok {
		Version = info.Main.Version
	}
}
