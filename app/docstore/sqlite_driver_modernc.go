//go:build nativesqlite
// +build nativesqlite

package docstore

import (
	_ "modernc.org/sqlite"
)

const SQLiteDriverName = "sqlite"
