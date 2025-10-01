package dictionary

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
)

type SearchParams struct {
	Query     string
	TextQuery string
	Tl        Transliteration
	Mode      SearchMode
}

type SearchSuggestionParams struct {
	PartialQuery string
	Tl           Transliteration
}

type SansDictionaryEntry struct {
	// Follow the structure of monier-williams dataset
}

type SearchSuggestion struct {
	IAST    string
	HK      string
	Nagari  string
	Preview string
}

type SearchSuggestionResponse struct {
	items []SearchSuggestion
}

type SearchResult struct {
	IAST    string
	HK      string
	Nagari  string
	Preview string
}

type SearchResponse struct {
	items []SearchResult
}
