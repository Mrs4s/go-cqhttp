//go:build amd64

package t544

import (
	"encoding/binary"
	"hash/crc32"
	"io"
)

var crc32Table = func() (tab crc32.Table) {
	f, err := cryptoZip.Open("crc32.bin")
	if err != nil {
		panic(err)
	}
	data, err := io.ReadAll(f)
	if err != nil {
		panic(err)
	}
	for i := range tab {
		tab[i] = binary.LittleEndian.Uint32(data[i*4 : (i+1)*4])
	}
	return
}()

//go:noescape
func tencentCrc32(tab *crc32.Table, b []byte) uint32

//go:noescape
func sub_a([]byte, []uint32)

//go:noescape
func sub_b([]byte, []uint32)

//go:noescape
func sub_c(*[16][16]byte, []byte)

//go:noescape
func sub_d(*[16]byte, []byte)

//go:noescape
func sub_e(*[256][6]byte, []byte)

//go:noescape
func sub_f(*[16]byte, *[15]uint32, *[16][16]byte) (w [44]uint32)

//go:noescape
func sub_aa(int, *[16][2][16][16]byte, *[16]byte, []byte) byte

// transformInner see com/tencent/mobileqq/dt/model/FEBound
//
//go:noescape
func transformInner(*[0x15]byte, *[32][16]byte)

//go:noescape
func initState(*state, []byte, []byte, uint64)

func (c *state) init(key []byte, data []byte, counter uint64, nr uint8) {
	c.nr = nr
	c.p = 0
	initState(c, key, data, counter)
}

//go:noescape
func refreshState(c *state)
