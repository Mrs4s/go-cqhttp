package t544

import (
	"crypto/rand"
	"encoding/hex"
	"testing"
)

func TestT544(t *testing.T) {
	r := hex.EncodeToString(sign(0, []byte{}))
	if r != "0c05d28b405bce1595c70ffa694ff163d4b600f229482e07de32c8000000003525382c00000000" {
		t.Fatal(r)
	}
}

func TestCrash(t *testing.T) {
	brand := make([]byte, 4096)
	for i := 1; i <= 1024; i++ {
		rand.Reader.Read(brand)
		sign(123, brand)
	}
}
