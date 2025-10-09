package common

import (
	"strconv"
	"strings"
)

type Language string

const (
	Sanskrit Language = "sanskrit"
	English  Language = "english"
)

// Accent marks will be ignored for search
type Transliteration string

const (
	TlIAST   Transliteration = "iast"
	TlHK     Transliteration = "hk"
	TlNagari Transliteration = "dn"
	TlSLP1   Transliteration = "slp1"
)

type SearchMode string

const (
	SearchExact  SearchMode = "exact"
	SearchRegex  SearchMode = "re"
	SearchPrefix SearchMode = "prefix"
	SearchFuzzy  SearchMode = "fuzzy"
)

func PathToString(path []int) string {
	var parts []string
	for _, p := range path {
		parts = append(parts, strconv.Itoa(p))
	}
	return strings.Join(parts, ".")
}
