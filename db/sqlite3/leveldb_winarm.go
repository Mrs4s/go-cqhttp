//go:build windows && (arm || arm64)
// +build windows
// +build arm arm64

package sqlite3

import _ "github.com/Mrs4s/go-cqhttp/db/leveldb" // 切换到 leveldb
