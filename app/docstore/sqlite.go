package docstore

import (
	"database/sql"
	"log/slog"
	"path/filepath"
)

// NewSQLiteDB creates a new SQLite DB connection.
func NewSQLiteDB(dataDir string, readonly bool) (*sql.DB, error) {
	dbPath := filepath.Join(dataDir, "dhee.db")
	if readonly {
		dbPath = dbPath + "?mode=ro"
	}
	slog.Info("opening SQLite DB", "dbPath", dbPath)
	db, err := sql.Open(SQLiteDriverName, dbPath)
	if err != nil {
		return nil, err
	}
	return db, nil
}
