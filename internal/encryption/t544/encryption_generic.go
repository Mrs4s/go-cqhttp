//go:build amd64

package t544

import (
	"encoding/binary"
	"io"
)

type encryptionData struct {
	tableA [16][2][16][16]byte
	tableB [16][16]byte
	tableC [256][6]byte
	tableD [16]byte
	tableE [16]byte
	tableF [15]uint32
}

type state struct {
	state    [16]uint32 // 16
	orgstate [16]uint32 // 16
	nr       uint8
	p        uint8
}

var crypto = encryptionData{
	tableA: readData[[16][2][16][16]byte]("table_a.bin"),
	tableB: readData[[16][16]byte]("table_b.bin"),
	tableC: readData[[256][6]byte]("table_c.bin"),
	tableD: readData[[16]byte]("table_d.bin"),
	tableE: readData[[16]byte]("table_e.bin"),
	tableF: func() (tab [15]uint32) {
		f, err := cryptoZip.Open("table_f.bin")
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
	}(),
}

func (e *encryptionData) tencentEncryptB(p1 []byte, p2 []uint32) {
	const c = 10
	for r := 0; r < 9; r++ {
		sub_d(&e.tableD, p1)
		sub_b(p1, p2[r*4:(r+1)*4])
		sub_c(&e.tableB, p1)
		sub_e(&e.tableC, p1)
	}
	sub_d(&e.tableD, p1)
	sub_b(p1, p2[(c-1)*4:c*4])
	sub_c(&e.tableB, p1)
	sub_a(p1, p2[c*4:(c+1)*4])
}

func (e *encryptionData) tencentEncryptionB(c []byte, m []byte) (out [0x15]byte) {
	var buf [16]byte
	w := sub_f(&e.tableE, &e.tableF, &e.tableB)

	for i := range out {
		if (i & 0xf) == 0 {
			copy(buf[:], c)
			e.tencentEncryptB(buf[:], w[:])
			for j := 15; j >= 0; j-- {
				c[j]++
				if c[j] != 0 {
					break
				}
			}
		}
		out[i] = sub_aa(i, &e.tableA, &buf, m)
	}

	return
}

func tencentEncryptionA(input, key, data []byte) {
	var s state
	s.init(key, data, 0, 20)
	s.encrypt(input)
}

func (c *state) encrypt(data []byte) {
	bp := 0
	dataLen := uint32(len(data))
	for dataLen > 0 {
		if c.p == 0 {
			for i := uint8(0); i < c.nr; i += 2 {
				sub_ad(c.state[:])
			}
			for i := 0; i < 16; i++ {
				c.state[i] += c.orgstate[i]
			}
		}
		var sb [16 * 4]byte
		for i, v := range c.state {
			binary.LittleEndian.PutUint32(sb[i*4:(i+1)*4], v)
		}
		for c.p != 64 && dataLen != 0 {
			data[bp] ^= sb[c.p]
			c.p++
			bp++
			dataLen--
		}
		if c.p >= 64 {
			c.p = 0
			c.orgstate[12]++
			c.state = c.orgstate
		}
	}
}
