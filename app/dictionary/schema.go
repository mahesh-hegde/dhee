package dictionary

import "github.com/mahesh-hegde/dhee/app/common"

type SearchParams struct {
	Query     string
	Tl        common.Transliteration
	Mode      common.SearchMode
	TextQuery string
}

type SuggestParams struct {
	PartialQuery string
	Tl           common.Transliteration
}

type DictSearchSuggestion struct {
	IAST    string
	HK      string
	Nagari  string
	Preview string
}

type Suggestions struct {
	Items []DictSearchSuggestion
}

type DictSearchResult struct {
	IAST    string
	HK      string
	Nagari  string
	Preview string
}

type SearchResults struct {
	Items []DictSearchResult
}

type Cognate struct {
	Language string `json:"language"`
	Word     string `json:"word"`
}

type LexCat struct {
	LexID       string `json:"lex_id,omitempty"`
	Stem        string `json:"stem,omitempty"`
	RootClass   string `json:"root_class,omitempty"`
	IsLoan      bool   `json:"is_loan,omitempty"`
	InflictType string `json:"inflict_type,omitempty"`
}

type Verb struct {
	VerbType  string   `json:"verb_type,omitempty"`
	VerbClass int      `json:"verb_class,omitempty"`
	Pada      string   `json:"pada,omitempty"`  // Either "A" or "P" if known, for Atmanepada and Parasmaipada respectively.
	Parse     []string `json:"parse,omitempty"` // [prefix, root]
}

// DictionaryEntry represents the processed dictionary entry
type DictionaryEntry struct {
	// Not written to JSONL data but computed based on which file we read,
	// and stored in DB to distinguish.
	DictName       string              `json:"dict_name,omitempty"`
	Word           string              `json:"word"`
	HTag           string              `json:"htag"`
	Id             string              `json:"id"`
	IAST           string              `json:"iast,omitempty"`
	Devanagari     string              `json:"devanagari,omitempty"`
	Variants       []string            `json:"variants,omitempty"`
	PrintedPageNum string              `json:"print_page"`
	Cognates       []string            `json:"cognates,omitempty"`
	LitRefs        []string            `json:"lit_refs,omitempty"`
	LexicalGender  string              `json:"lexical_gender,omitempty"`
	Body           DictionaryEntryBody `json:"body"`
	HomonymNumber  int                 `json:"homonym_number,omitempty"`
	Stem           string              `json:"stem,omitempty"`
	IsAnimalName   bool                `json:"is_animal_name,omitempty"`
	IsPlantName    bool                `json:"is_plant_name,omitempty"`
	LexCat         LexCat              `json:"lexcat,omitzero"`
	Verb           Verb                `json:"verb,omitzero"`
}

// Type implements mapping.Classifier.
func (d *DictionaryEntry) Type() string {
	return "dictionary_entry"
}

type DictionaryEntryBody struct {
	Plain  string `json:"plain"`
	Markup string `json:"-"`
}

type DictDefn struct {
	// A name slug used in URLs. Eg: monier-williams
	Name string `json:"name"`
	// A human readable name used in titles. Eg: "Monier-Williams Sanskrit-English dictionary"
	ReadableName   string                 `json:"readable_name"`
	SourceLanguage common.Language        `json:"source_language"`
	TargetLanguage common.Language        `json:"target_language"`
	WordEncoding   common.Transliteration `json:"word_encoding"`

	// File with entries encoded as JSONL
	DataFile string `json:"data_file"`
}
