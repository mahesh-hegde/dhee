package dictionary

import (
	"context"
	"log/slog"

	"github.com/blevesearch/bleve/v2"
	"github.com/mahesh-hegde/dhee/app/common"
	"github.com/mahesh-hegde/dhee/app/config"
	"github.com/mahesh-hegde/dhee/app/transliteration"
)

type DictionaryService struct {
	store          DictStore
	conf           *config.DheeConfig
	transliterator *transliteration.Transliterator
}

// GetEntries takes a list of words and returns the full dictionary
// data about those words
func (s *DictionaryService) GetEntries(ctx context.Context, dictionaryName string, words []string, tl common.Transliteration) (map[string]DictionaryEntry, error) {
	slp1Words := make([]string, len(words))
	for i, word := range words {
		slp1Words[i] = word
		if tl != common.TlSLP1 {
			slp1Word, err := s.transliterator.Convert(word, tl, common.TlSLP1)
			if err != nil {
				slog.Warn("transliteration failed for word", "word", word, "err", err)
			} else {
				slp1Words[i] = slp1Word
			}
		}
	}

	results, err := s.store.Get(ctx, dictionaryName, slp1Words)
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
	slp1Word, err := s.transliterator.Convert(partialWord, tl, common.TlSLP1)
	if err != nil {
		slog.Warn("transliteration failed for suggestion", "word", partialWord, "err", err)
		slp1Word = partialWord
	}

	return s.store.Suggest(ctx, dictName, SuggestParams{
		PartialQuery: slp1Word,
	})
}

func (s *DictionaryService) Search(ctx context.Context, dictionaryName string, searchParams SearchParams) (SearchResults, error) {
	slp1Query, err := s.transliterator.Convert(searchParams.Query, searchParams.Tl, common.TlSLP1)
	if err != nil {
		slog.Warn("transliteration failed for search", "query", searchParams.Query, "err", err)
		slp1Query = searchParams.Query
	}
	searchParams.Query = slp1Query

	return s.store.Search(ctx, dictionaryName, searchParams)
}

func (s *DictionaryService) Related(ctx context.Context, dictName string, word string) (SearchResults, error) {
	// Assuming the input 'word' for related is already in SLP1 from a dictionary entry.
	return s.store.Related(ctx, dictName, word)
}

func NewDictionaryService(index bleve.Index, conf *config.DheeConfig, transliterator *transliteration.Transliterator) *DictionaryService {
	return &DictionaryService{
		store:          NewBleveDictStore(index, conf),
		conf:           conf,
		transliterator: transliterator,
	}
}
