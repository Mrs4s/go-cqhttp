//go:build amd64

package t544

func cpuid(op uint32) (eax, ebx, ecx, edx uint32)

var canusesse2 = func() bool {
	_, _, _, d := cpuid(1)
	return d&(1<<26) > 0
}()
