//go:build native_sqlite
// +build native_sqlite

package docstore

import (
	"database/sql/driver"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/patrickmn/go-cache"
	sqlite "modernc.org/sqlite"
)

var regexCache = cache.New(2*time.Minute, 5*time.Minute)

func init() {
	sqlite.MustRegisterDeterministicScalarFunction(
		"regexp",
		2,
		func(ctx *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
			var s1 string
			var s2 string

			switch arg0 := args[0].(type) {
			case string:
				s1 = arg0
			default:
				return nil, errors.New("expected argv[0] to be text")
			}

			switch arg1 := args[1].(type) {
			case string:
				s2 = arg1
			default:
				return nil, errors.New("expected argv[1] to be text")
			}

			re := s1
			if !strings.HasPrefix(s1, "(?s)") {
				re = "(?s)" + s1
			}

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
			return compiledRe.MatchString(s2), nil
		},
	)
}

const SQLiteDriverName = "sqlite"
