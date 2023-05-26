//go:build amd64

package t544

import (
	"crypto/md5"
	"crypto/rc4"
	"encoding/binary"
	"math/rand"

	"github.com/Mrs4s/go-cqhttp/internal/encryption"
)

const (
	keyTable = "$%&()+,-456789:?ABCDEEFGHIJabcdefghijkopqrstuvwxyz"
	table2   = "!#$%&)+.0123456789:=>?@ABCDEFGKMNabcdefghijkopqrst"
)

var (
	magic = uint64(0x6EEDCF0DC4675540)
	key1  = [8]byte{'a', '$', '(', 'e', 'T', '7', '*', '@'}
	key2  = [8]byte{'&', 'O', '9', '!', '>', '6', 'X', ')'}
)

func init() {
	encryption.T544Signer["8.9.35.10440"] = sign
	encryption.T544Signer["8.9.38.10545"] = sign
}

// sign t544 algorithm
// special thanks to the anonymous contributor who provided the algorithm
func sign(curr int64, input []byte) []byte {
	var crcData [0x15]byte
	curr %= 1000000
	binary.BigEndian.PutUint32(crcData[:4], uint32(curr))
	input = append(input, crcData[:4]...)
	var kt [4 + 32 + 4]byte
	r := rand.New(rand.NewSource(curr))
	for i := 0; i < 2; i++ {
		kt[i] = keyTable[r.Int()%0x32] + 50
	}
	kt[2] = kt[1] + 20
	kt[3] = kt[2] + 20
	key3 := kt[4 : 4+10]
	k3calc := key3[2:10]
	copy(k3calc, key1[:4])
	for i := 0; i < 4; i++ {
		k3calc[4+i] = key2[i] ^ kt[i]
	}
	key3[0], key3[1] = k3calc[6], k3calc[7]
	key3 = key3[:8]
	k3calc[6], k3calc[7] = 0, 0
	rc4Cipher, _ := rc4.NewCipher(key3)
	rc4Cipher.XORKeyStream(key3, key3)
	binary.LittleEndian.PutUint64(crcData[4:4+8], magic)
	tencentEncryptionA(input, kt[4:4+32], crcData[4:4+8])
	result := md5.Sum(input)
	crcData[2] = 1
	crcData[4] = 1
	copy(crcData[5:9], kt[:4])
	binary.BigEndian.PutUint32(crcData[9:13], uint32(curr))
	copy(crcData[13:], result[:8])
	calcCrc := tencentCrc32(&crc32Table, crcData[2:])
	binary.LittleEndian.PutUint32(kt[4+32:4+32+4], calcCrc)
	crcData[0] = kt[4+32]
	crcData[1] = kt[4+32+3]
	nonce := uint32(r.Int() ^ r.Int() ^ r.Int())
	on := kt[:16]
	binary.BigEndian.PutUint32(on[:4], nonce)
	copy(on[4:8], on[:4])
	copy(on[8:16], on[:8])
	ts.transformEncode(&crcData)
	encryptedData := crypto.tencentEncryptionB(on, crcData[:])
	ts.transformDecode(&encryptedData)
	output := kt[:39]
	output[0] = 0x0C
	output[1] = 0x05
	binary.BigEndian.PutUint32(output[2:6], nonce)
	copy(output[6:27], encryptedData[:])
	binary.LittleEndian.PutUint32(output[27:31], 0)
	output[31] = table2[r.Int()%0x32]
	output[32] = table2[r.Int()%0x32]
	addition := r.Int() % 9
	for addition&1 == 0 {
		addition = r.Int() % 9
	}
	output[33] = output[31] + byte(addition)
	output[34] = output[32] + byte(9-addition) + 1
	binary.LittleEndian.PutUint32(output[35:39], 0)
	return output
}
