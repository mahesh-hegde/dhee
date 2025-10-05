package dictionary

import (
	"context"

	"github.com/blevesearch/bleve/v2"
	"github.com/mahesh-hegde/dhee/app/config"
)

type DictStore interface {
	// Add simply batch-adds all the dictionary entries to the
	// dictionary, making sure they have proper dictName set.
	Add(ctx context.Context, dictName string, es []DictionaryEntry) error

	// Get returns one more dictionary entries per word
	Get(ctx context.Context, dictName string, words []string) (map[string][]DictionaryEntry, error)

	// Search does exact-word/prefix/regex/fuzzy search according to searchParams. transliteration
	// is ignored for now and assumed to be SLP1 (same as `word` field in the data).
	Search(ctx context.Context, dictName string, s SearchParams) SearchResults

	// Suggest returns suggestions of the word starting with s.PartialQuery. For now, transliteration is
	// ignored and assumed to be SLP1
	Suggest(ctx context.Context, dictName string, s SuggestParams) Suggestions

	// Related returns the entries starting with `{word}-` for now.
	Related(ctx context.Context, dictName string, word string) SearchResults
}

type BleveDictStore struct {
	idx  bleve.Index
	conf config.DheeConfig
}

// Add implements DictStore.
func (b *BleveDictStore) Add(ctx context.Context, dictName string, es []DictionaryEntry) error {
	panic("unimplemented")
}

// Get implements DictStore.
func (b *BleveDictStore) Get(ctx context.Context, dictName string, words []string) (map[string][]DictionaryEntry, error) {
	panic("unimplemented")
}

// Related implements DictStore.
func (b *BleveDictStore) Related(ctx context.Context, dictName string, word string) SearchResults {
	panic("unimplemented")
}

// Search implements DictStore.
func (b *BleveDictStore) Search(ctx context.Context, dictName string, s SearchParams) SearchResults {
	panic("unimplemented")
}

// Suggest implements DictStore.
func (b *BleveDictStore) Suggest(ctx context.Context, dictName string, s SuggestParams) Suggestions {
	panic("unimplemented")
}

var _ DictStore = &BleveDictStore{}

func NewBleveDictStore(idx bleve.Index, conf config.DheeConfig) *BleveDictStore {
	return &BleveDictStore{idx: idx, conf: conf}
}
