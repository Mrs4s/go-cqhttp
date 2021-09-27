package btree

import "unsafe"

func assert(cond bool) {
	if !cond {
		panic("assert failed!")
	}
}

// power2 returns a value that is greater or equal to 'val' and is power-of-two.
func power2(val int) int {
	i := 1
	for i < val {
		i <<= 1
	}
	return i
}

// helpers for sha1

func cmp(a, b *byte) uint64 {
	pa, pb := unsafe.Pointer(a), unsafe.Pointer(b)
	if *(*uint64)(pa) != *(*uint64)(pb) {
		return *(*uint64)(pa) - *(*uint64)(pb)
	}
	pa, pb = unsafe.Add(pa, 8), unsafe.Add(pb, 8)
	if *(*uint64)(pa) != *(*uint64)(pb) {
		return *(*uint64)(pa) - *(*uint64)(pb)
	}
	return uint64(*(*uint32)(unsafe.Add(pa, 8)) - *(*uint32)(unsafe.Add(pa, 8)))
}

func copysha1(dst *byte, src *byte) {
	pa, pb := unsafe.Pointer(dst), unsafe.Pointer(src)
	*(*[sha1Size]byte)(pa) = *(*[sha1Size]byte)(pb)
}

func resetsha1(sha1 *byte) {
	p := unsafe.Pointer(sha1)
	*(*[sha1Size]byte)(p) = [sha1Size]byte{}
}
