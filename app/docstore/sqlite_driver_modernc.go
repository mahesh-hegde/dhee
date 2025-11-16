//go:build !cgo
// +build !cgo

package docstore

import (
	_ "modernc.org/sqlite"
)

const SQLiteDriverName = "sqlite"
