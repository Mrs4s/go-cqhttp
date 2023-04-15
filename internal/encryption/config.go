package encryption

// T544Getter get T544
type T544Getter func(int64, []byte) []byte

// T544Signer map[version]func
var T544Signer = map[string]T544Getter{}
