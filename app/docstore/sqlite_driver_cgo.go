//go:build !nativesqlite
// +build !nativesqlite

package docstore

import (
	_ "github.com/mattn/go-sqlite3"
)

const SQLiteDriverName = "sqlite3"
