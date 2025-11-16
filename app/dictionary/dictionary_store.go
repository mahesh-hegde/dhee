package dictionary

import (
	"context"
	"encoding/json"
	"log/slog"
)

func getAllVariants(entry *DictionaryEntry) []string {
	res := make([]string, 0)
	for _, meaning := range entry.Meanings {
		res = append(res, meaning.Variants...)
	}
	return res
}

func getAllLitRefs(entry *DictionaryEntry) []string {
	res := make([]string, 0)
	for _, meaning := range entry.Meanings {
		res = append(res, meaning.LitRefs...)
	}
	return res
}

type DictStore interface {
	Init() error
	// Add simply batch-adds all the dictionary entries to the
	// dictionary, making sure they have proper dictName set.
	Add(ctx context.Context, dictName string, es []DictionaryEntry) error

	// Get returns one more dictionary entries per word
	Get(ctx context.Context, dictName string, words []string) (map[string]DictionaryEntry, error)

	// Search does exact-word/prefix/regex/fuzzy search according to searchParams. transliteration
	// is ignored for now and assumed to be SLP1 (same as `word` field in the data).
	Search(ctx context.Context, dictName string, s SearchParams) (SearchResults, error)

	// Suggest returns suggestions of the word starting with s.PartialQuery. For now, transliteration is
	// ignored and assumed to be SLP1
	Suggest(ctx context.Context, dictName string, s SuggestParams) (Suggestions, error)

	// Related returns the entries starting with `{word}-` for now.
	Related(ctx context.Context, dictName string, word string) (SearchResults, error)
}

func prepareDictEntryForDb(e *DictionaryEntry) DictionaryEntryInDB {
	// Marshal full entry
	entryJSON, err := json.Marshal(e)
	if err != nil {
		slog.Error("unexpected error", "err", err)
		panic(err)
	}

	bodyText := []string{}
	for _, meaning := range e.Meanings {
		bodyText = append(bodyText, meaning.Body.Plain)
	}

	return DictionaryEntryInDB{
		DictName: e.DictName,
		Word:     e.Word,
		Entry:    string(entryJSON),
		Variants: getAllVariants(e),
		LitRefs:  getAllLitRefs(e),
		BodyText: bodyText,
	}
}
