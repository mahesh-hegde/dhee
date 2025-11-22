//go:build !native_sqlite
// +build !native_sqlite

package docstore

import (
	"database/sql"
	"regexp"
	"time"

	"github.com/mattn/go-sqlite3"
	_ "github.com/mattn/go-sqlite3"
	"github.com/patrickmn/go-cache"
)

var regexCache = cache.New(2*time.Minute, 5*time.Minute)

func regexMatch(re, s string) (bool, error) {
	var compiledRe *regexp.Regexp
	var err error

	if reFromCache, found := regexCache.Get(re); found {
		compiledRe = reFromCache.(*regexp.Regexp)
	} else {
		compiledRe, err = regexp.Compile(re)
		if err != nil {
			return false, err
		}
		regexCache.Set(re, compiledRe, cache.DefaultExpiration)
	}

	return compiledRe.MatchString(s), nil
}

const SQLiteDriverName = "sqlite3_extended"

func init() {
	sql.Register(
		SQLiteDriverName,
		&sqlite3.SQLiteDriver{
			ConnectHook: func(conn *sqlite3.SQLiteConn) error {
				return conn.RegisterFunc("regexp", regexMatch, true)
			},
		})
}
