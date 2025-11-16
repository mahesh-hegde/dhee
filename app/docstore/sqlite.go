package docstore

import (
	"database/sql"
	"path/filepath"
)

// NewSQLiteDB creates a new SQLite DB connection.
func NewSQLiteDB(dataDir string, readonly bool) (*sql.DB, error) {
	dbPath := filepath.Join(dataDir, "dhee.db")
	if readonly {
		dbPath = dbPath + "?mode=ro"
	}
	db, err := sql.Open(SQLiteDriverName, dbPath)
	if err != nil {
		return nil, err
	}
	return db, nil
}
