//go:build !native_sqlite
// +build !native_sqlite

package docstore

import (
	_ "github.com/mattn/go-sqlite3"
)

const SQLiteDriverName = "sqlite3"
