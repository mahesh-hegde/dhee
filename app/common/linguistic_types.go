package common

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
)

type SearchMode string

const (
	SearchExact  SearchMode = "exact"
	SearchRegex  SearchMode = "re"
	SearchPrefix SearchMode = "prefix"
	SearchFuzzy  SearchMode = "fuzzy"
)
