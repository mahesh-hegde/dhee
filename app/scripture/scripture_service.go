package scripture

import (
	"context"
	"fmt"
	"strings"

	"github.com/blevesearch/bleve/v2"
	"github.com/mahesh-hegde/dhee/app/config"
	"github.com/mahesh-hegde/dhee/app/dictionary"
)

type ScriptureService struct {
	ds    dictionary.DictStore
	store ExcerptStore
	conf  config.DheeConfig
}

type ExcerptWithWords struct {
	E     Excerpt
	Words map[string]dictionary.DictionaryEntry
}

// Get returns the excerpts given by paths. If any of the excerpts could not be found, it returns an error.
//
// Get also batch-fetches the dictionary words for surfaces,
// and lemmas (stripping `-` at end for lemmas). In this process, we expect most entries do not exist
// in the dictionary. We return only those that were found in the batch search.
func (s *ScriptureService) Get(ctx context.Context, paths []QualifiedPath) ([]ExcerptWithWords, error) {
	excerpts := s.store.Get(ctx, paths)
	if len(excerpts) != len(paths) {
		// TODO: Return a more specific error
		return nil, fmt.Errorf("could not find all excerpts")
	}

	// Collect all words to fetch from the dictionary
	wordsToFetch := make(map[string]bool)
	for _, e := range excerpts {
		for _, g := range e.Glossings {
			for _, gl := range g {
				wordsToFetch[gl.Surface] = true
				lemma := strings.TrimSuffix(gl.Lemma, "-")
				wordsToFetch[lemma] = true
			}
		}
	}

	var words []string
	for w := range wordsToFetch {
		words = append(words, w)
	}

	// Fetch dictionary entries. We assume the first dictionary is the one we want.
	// TODO: Make this configurable.
	dictName := s.conf.Dictionaries[0].Name
	dictEntries, err := s.ds.Get(ctx, dictName, words)
	if err != nil {
		return nil, fmt.Errorf("failed to get dictionary entries: %w", err)
	}

	// Map words to their dictionary entries for quick lookup
	wordMap := make(map[string]dictionary.DictionaryEntry)
	for _, entries := range dictEntries {
		if len(entries) > 0 {
			// If multiple entries exist, just take the first one for now.
			wordMap[entries[0].Word] = entries[0]
		}
	}

	// Combine excerpts with their word meanings
	var result []ExcerptWithWords
	for _, e := range excerpts {
		ew := ExcerptWithWords{
			E:     e,
			Words: make(map[string]dictionary.DictionaryEntry),
		}
		for _, g := range e.Glossings {
			for _, gl := range g {
				if entry, ok := wordMap[gl.Surface]; ok {
					ew.Words[gl.Surface] = entry
				}
				lemma := strings.TrimSuffix(gl.Lemma, "-")
				if entry, ok := wordMap[lemma]; ok {
					ew.Words[lemma] = entry
				}
			}
		}
		result = append(result, ew)
	}

	return result, nil
}

// Search returns upto 100 Excerpts which match the search according to search parameters.
func (s *ScriptureService) Search(ctx context.Context, search SearchParams) ([]Excerpt, error) {
	// If no scriptures are specified, search in all of them.
	var scriptures []string
	if len(search.Scriptures) == 0 {
		for _, sc := range s.conf.Scriptures {
			scriptures = append(scriptures, sc.Name)
		}
	} else {
		scriptures = search.Scriptures
	}

	excerpts := s.store.Search(ctx, scriptures, search)
	return excerpts, nil
}

func NewScriptureService(index bleve.Index, conf config.DheeConfig) *ScriptureService {
	return &ScriptureService{
		ds:    dictionary.NewBleveDictStore(index, conf),
		store: NewBleveExcerptStore(index, conf),
		conf:  conf,
	}
}
