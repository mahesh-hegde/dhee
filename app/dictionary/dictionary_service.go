package dictionary

import (
	"context"
	"log/slog"

	"github.com/blevesearch/bleve/v2"
	"github.com/mahesh-hegde/dhee/app/common"
	"github.com/mahesh-hegde/dhee/app/config"
)

type DictionaryService struct {
	store DictStore
	conf  config.DheeConfig
}

// GetEntries takes a list of words and returns the full dictionary
// data about those words
func (s *DictionaryService) GetEntries(ctx context.Context, dictionaryName string, words []string, tl common.Transliteration) (map[string]DictionaryEntry, error) {
	results, err := s.store.Get(ctx, dictionaryName, words)
	if err != nil {
		slog.Error("failed to get entries from store", "err", err)
		return nil, err
	}

	// The store returns a slice of entries for each word. For the service layer,
	// we'll just return the first one for now.
	finalResults := make(map[string]DictionaryEntry)
	for word, entries := range results {
		if len(entries) > 0 {
			finalResults[word] = entries[0]
		}
	}
	return finalResults, nil
}

func (s *DictionaryService) Suggest(ctx context.Context, dictName string, partialWord string, tl common.Transliteration) (Suggestions, error) {
	// TODO: Use transliteration
	return s.store.Suggest(ctx, dictName, SuggestParams{
		PartialQuery: partialWord,
	})
}

func (s *DictionaryService) Search(ctx context.Context, dictionaryName string, searchParams SearchParams) (SearchResults, error) {
	// TODO: Use transliteration
	return s.store.Search(ctx, dictionaryName, searchParams)
}

func (s *DictionaryService) Related(ctx context.Context, dictName string, word string) (SearchResults, error) {
	// TODO: Use transliteration
	return s.store.Related(ctx, dictName, word)
}

func NewDictionaryService(index bleve.Index, conf config.DheeConfig) *DictionaryService {
	return &DictionaryService{
		store: NewBleveDictStore(index, conf),
		conf:  conf,
	}
}
