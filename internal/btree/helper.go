package btree

import (
	"io"
	"reflect"
	"unsafe"
)

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

// helpers for hash

func cmp(a, b *byte) int64 {
	pa, pb := unsafe.Pointer(a), unsafe.Pointer(b)
	if *(*uint64)(pa) != *(*uint64)(pb) {
		return int64(*(*uint64)(pa) - *(*uint64)(pb))
	}
	pa, pb = unsafe.Add(pa, 8), unsafe.Add(pb, 8)
	return int64(*(*uint64)(pa) - *(*uint64)(pb))
}

func copyhash(dst *byte, src *byte) {
	pa, pb := unsafe.Pointer(dst), unsafe.Pointer(src)
	*(*[hashSize]byte)(pa) = *(*[hashSize]byte)(pb)
}

func resethash(sha1 *byte) {
	p := unsafe.Pointer(sha1)
	*(*[hashSize]byte)(p) = [hashSize]byte{}
}

// reading table

func read32(r io.Reader) (int32, error) {
	var b = make([]byte, 4)
	_, err := r.Read(b)
	if err != nil {
		return 0, err
	}
	return *(*int32)(unsafe.Pointer(&b[0])), nil
}

func readTable(r io.Reader, t *table) error {
	buf := make([]byte, tableStructSize)
	_, err := r.Read(buf)
	if err != nil {
		return err
	}
	*t = *(*table)(unsafe.Pointer(&buf[0]))
	return nil
}

func readSuper(r io.Reader, s *super) error {
	buf := make([]byte, superSize)
	_, err := r.Read(buf)
	if err != nil {
		return err
	}
	*s = *(*super)(unsafe.Pointer(&buf[0]))
	return nil
}

// write table

func write32(w io.Writer, t int32) error {
	var p []byte
	ph := (*reflect.SliceHeader)(unsafe.Pointer(&p))
	ph.Data = uintptr(unsafe.Pointer(&t))
	ph.Len = 4
	ph.Cap = 4
	_, err := w.Write(p)
	return err
}

func writeTable(w io.Writer, t *table) error {
	var p []byte
	ph := (*reflect.SliceHeader)(unsafe.Pointer(&p))
	ph.Data = uintptr(unsafe.Pointer(t))
	ph.Len = tableStructSize
	ph.Cap = tableStructSize
	_, err := w.Write(p)
	return err
}

func writeSuper(w io.Writer, s *super) error {
	var p []byte
	ph := (*reflect.SliceHeader)(unsafe.Pointer(&p))
	ph.Data = uintptr(unsafe.Pointer(s))
	ph.Len = superSize
	ph.Cap = superSize
	_, err := w.Write(p)
	return err
}
