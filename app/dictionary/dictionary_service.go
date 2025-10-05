package dictionary

import (
	"context"

	"github.com/mahesh-hegde/dhee/app/common"
)

type DictionaryService struct{}

// GetEntries takes a list of words and returns the full dictionary
// data about those words
func (s *DictionaryService) GetEntries(ctx context.Context, dictionaryName string, words []string, tl common.Transliteration) map[string]DictionaryEntry {
	return map[string]DictionaryEntry{}
}

func (s *DictionaryService) Suggest(ctx context.Context, dictName string, partialWord []string, tl common.Transliteration) Suggestions {
	return Suggestions{}
}

func (s *DictionaryService) Search(ctx context.Context, dictionaryName string, SearchParams []string) SearchResults {
	return SearchResults{}
}

func (s *DictionaryService) Related(ctx context.Context, dictName string, word string) SearchResults {
	return SearchResults{}
}
