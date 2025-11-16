//go:build cgo
// +build cgo

package docstore

import (
	_ "github.com/mattn/go-sqlite3"
)

const SQLiteDriverName = "sqlite3"
