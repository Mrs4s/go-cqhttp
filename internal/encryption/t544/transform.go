//go:build amd64

package t544

type transformer struct {
	encode [32][16]byte
	decode [32][16]byte
}

func (ts *transformer) transformEncode(bArr *[0x15]byte) {
	transformInner(bArr, &ts.encode)
}

func (ts *transformer) transformDecode(bArr *[0x15]byte) {
	transformInner(bArr, &ts.decode)
}

var ts = transformer{
	encode: readData[[32][16]byte]("encode.bin"),
	decode: readData[[32][16]byte]("decode.bin"),
}
