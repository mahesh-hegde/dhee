//go:build native_sqlite
// +build native_sqlite

package docstore

import (
	_ "modernc.org/sqlite"
)

const SQLiteDriverName = "sqlite"
