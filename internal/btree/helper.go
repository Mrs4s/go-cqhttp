package btree

import (
	"encoding/binary"
	"io"
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

// helpers for sha1

func cmp(a, b *byte) int64 {
	pa, pb := unsafe.Pointer(a), unsafe.Pointer(b)
	if *(*uint64)(pa) != *(*uint64)(pb) {
		return int64(*(*uint64)(pa) - *(*uint64)(pb))
	}
	pa, pb = unsafe.Add(pa, 8), unsafe.Add(pb, 8)
	if *(*uint64)(pa) != *(*uint64)(pb) {
		return int64(*(*uint64)(pa) - *(*uint64)(pb))
	}
	return int64(*(*uint32)(unsafe.Add(pa, 8)) - *(*uint32)(unsafe.Add(pb, 8)))
}

func copysha1(dst *byte, src *byte) {
	pa, pb := unsafe.Pointer(dst), unsafe.Pointer(src)
	*(*[sha1Size]byte)(pa) = *(*[sha1Size]byte)(pb)
}

func resetsha1(sha1 *byte) {
	p := unsafe.Pointer(sha1)
	*(*[sha1Size]byte)(p) = [sha1Size]byte{}
}

// reading table

func read64(r io.Reader) (int64, error) {
	var b = make([]byte, 8)
	_, err := r.Read(b)
	if err != nil {
		return 0, err
	}
	return int64(binary.LittleEndian.Uint64(b)), nil
}

func read32(r io.Reader) (int32, error) {
	var b = make([]byte, 4)
	_, err := r.Read(b)
	if err != nil {
		return 0, err
	}
	return int32(binary.LittleEndian.Uint32(b)), nil
}

func readTable(r io.Reader, t *table) error {
	for i := 0; i < tableSize; i++ {
		err := readItem(r, &t.items[i])
		if err != nil {
			return err
		}
	}
	switch unsafe.Sizeof(0) {
	case 8:
		i, err := read64(r)
		t.size = int(i)
		return err
	case 4:
		i, err := read32(r)
		t.size = int(i)
		return err
	default:
		panic("unreachable")
	}
}

func readItem(r io.Reader, i *item) error {
	_, err := r.Read(i.sha1[:])
	if err != nil {
		return err
	}
	i.offset, err = read64(r)
	if err != nil {
		return err
	}
	i.child, err = read64(r)
	return err
}

func readSuper(r io.Reader, s *super) error {
	var err error
	if s.top, err = read64(r); err != nil {
		return err
	}
	if s.freeTop, err = read64(r); err != nil {
		return err
	}
	s.alloc, err = read64(r)
	return err
}

// write table

func write64(w io.Writer, i int64) error {
	var b = make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(i))
	_, err := w.Write(b)
	return err
}

func write32(w io.Writer, i int32) error {
	var b = make([]byte, 4)
	binary.LittleEndian.PutUint32(b, uint32(i))
	_, err := w.Write(b)
	return err
}

func writeTable(w io.Writer, t *table) error {
	for i := 0; i < tableSize; i++ {
		err := writeItem(w, &t.items[i])
		if err != nil {
			return err
		}
	}
	switch unsafe.Sizeof(0) {
	case 8:
		return write64(w, int64(t.size))
	case 4:
		return write32(w, int32(t.size))
	default:
		panic("unreachable")
	}
}

func writeItem(w io.Writer, i *item) error {
	if _, err := w.Write(i.sha1[:]); err != nil {
		return err
	}
	if err := write64(w, i.offset); err != nil {
		return err
	}
	return write64(w, i.child)
}

func writeSuper(w io.Writer, s *super) error {
	if err := write64(w, s.top); err != nil {
		return err
	}
	if err := write64(w, s.freeTop); err != nil {
		return err
	}
	return write64(w, s.alloc)
}
