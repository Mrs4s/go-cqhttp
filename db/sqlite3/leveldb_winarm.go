//go:build (windows && arm) || (windows && arm64)
// +build windows,arm windows,arm64

package sqlite3

import _ "github.com/Mrs4s/go-cqhttp/db/leveldb" // 切换到 leveldb
