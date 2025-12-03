package dictionary

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

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
func (s *DictionaryService) GetEntries(ctx context.Context, dictionaryName string, words []string, tl common.Transliteration) (DictionaryWordResponse, error) {
	slp1Words := make([]string, len(words))
	for i, word := range words {
		slp1Words[i] = word
		if tl != common.TlSLP1 {
			if tl == common.TlNagari {
				word = s.transliterator.FoldDevanagariAccents(word)
			}
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
		return DictionaryWordResponse{}, err
	}

	return DictionaryWordResponse{Words: results, Dictionary: s.conf.GetDictByName(dictionaryName)}, nil
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
	searchParams.OriginalQuery = searchParams.Query
	if searchParams.Tl == common.TlIAST {
		searchParams.Query = common.FoldAccents(searchParams.Query)
	}
	if searchParams.Tl == common.TlNagari {
		searchParams.Query = s.transliterator.FoldDevanagariAccents(searchParams.Query)
	}
	dict := s.conf.GetDictByName(dictionaryName)
	if dict == nil {
		return SearchResults{}, common.NewUserVisibleError(http.StatusNotFound, fmt.Sprintf("No such dictionary named %q", dictionaryName))
	}

	if searchParams.Mode != common.SearchTranslations {
		finalQuery, err := s.transliterator.Convert(searchParams.Query, searchParams.Tl, common.TlSLP1)
		if err != nil {
			slog.Warn("transliteration failed for search", "query", searchParams.Query, "err", err)
			finalQuery = searchParams.Query
		}
		searchParams.Query = finalQuery
	}

	res, err := s.store.Search(ctx, dictionaryName, searchParams)
	if err != nil {
		return SearchResults{}, err
	}
	for idx, itm := range res.Items {
		nagari, err := s.transliterator.Convert(itm.Word, common.TlSLP1, common.TlNagari)
		if err != nil {
			slog.Debug("error converting searched entry to devanagari", "word", itm.Word, "err", err)
		}
		res.Items[idx].Nagari = nagari
	}
	res.DictionaryReadableName = dict.ReadableName
	res.Params = searchParams
	return res, nil
}

func (s *DictionaryService) Related(ctx context.Context, dictName string, word string) (SearchResults, error) {
	// Assuming the input 'word' for related is already in SLP1 from a dictionary entry.
	return s.store.Related(ctx, dictName, word)
}

func NewDictionaryService(store DictStore, conf *config.DheeConfig, transliterator *transliteration.Transliterator) *DictionaryService {
	return &DictionaryService{
		store:          store,
		conf:           conf,
		transliterator: transliterator,
	}
}
