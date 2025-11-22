package dictionary

import (
	"github.com/mahesh-hegde/dhee/app/common"
	"github.com/mahesh-hegde/dhee/app/config"
)

type SearchParams struct {
	Query         string
	OriginalQuery string
	Tl            common.Transliteration
	Mode          common.SearchMode
	TextQuery     string
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
	IAST     string
	Word     string
	Nagari   string
	Previews []string
}

type SearchResults struct {
	DictionaryName string
	Items          []DictSearchResult
	Params         SearchParams
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

type Meaning struct {
	Word           string              `json:"word"`
	HTag           string              `json:"htag"`
	SId            string              `json:"id"`
	Variants       []string            `json:"variants,omitempty"`
	VariantsIAST   []string            `json:"variants_iast,omitempty"`
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
	// Other words referenced from this entry in SLP1 format
	Referenced []string `json:"referenced,omitempty"`
}

// DictionaryEntry represents the processed dictionary entry
type DictionaryEntry struct {
	// Not written to JSONL data but computed based on which file we read,
	// and stored in DB to distinguish.
	DictName string    `json:"dict_name,omitempty"`
	Word     string    `json:"word"`
	IAST     string    `json:"iast"`
	Meanings []Meaning `json:"meanings"`
}

type DictionaryEntryInDB struct {
	DictName string `json:"dict_name"`

	Word string `json:"word"`

	// JSON serialized entry on which no searching is performed
	Entry string `json:"e"`

	// variants from all entries collected together as keyword fields
	Variants []string `json:"variants"`

	// keyword fields
	LitRefs []string `json:"lit_refs"`

	// Body text indexed using standard english analyzer for full text searches
	BodyText []string `json:"body_text"`
}

// Type implements mapping.Classifier.
func (d *DictionaryEntryInDB) Type() string {
	return "dictionary_entry"
}

type DictionaryEntryBody struct {
	Plain  string `json:"plain"`
	Markup string `json:"-"`
}

type DictionaryWordResponse struct {
	Words      map[string]DictionaryEntry
	Dictionary *config.DictDefn
}
