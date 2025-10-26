package scripture

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/blevesearch/bleve/v2"
	"github.com/mahesh-hegde/dhee/app/common"
	"github.com/mahesh-hegde/dhee/app/config"
	"github.com/mahesh-hegde/dhee/app/dictionary"
	"github.com/mahesh-hegde/dhee/app/transliteration"
)

type ExcerptService struct {
	ds             dictionary.DictStore
	store          ExcerptStore
	conf           *config.DheeConfig
	transliterator *transliteration.Transliterator
	scriptureMap   map[string]config.ScriptureDefn
}

type ExcerptWithWords struct {
	Excerpt
	Words map[string][]dictionary.DictionaryEntry
}

// Get returns the excerpts given by paths. If any of the excerpts could not be found, it returns an error.
//
// Get also batch-fetches the dictionary words for surfaces,
// and lemmas (stripping `-` at end for lemmas). In this process, we expect most entries do not exist
// in the dictionary. We return only those that were found in the batch search.
func (s *ExcerptService) Get(ctx context.Context, paths []QualifiedPath) (*ExcerptTemplateData, error) {
	if len(paths) == 0 {
		return nil, fmt.Errorf("specify a scripture and path")
	}
	excerpts, err := s.store.Get(ctx, paths)
	if err != nil {
		slog.Error("error retrieving excerpts", "err", err)
		return nil, fmt.Errorf("failed to retrieve")
	}

	if len(excerpts) != len(paths) {
		slog.Error("not all excerpts could be found", "err", err)
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
	dictName := s.conf.DefaultDict
	dictEntries, err := s.ds.Get(ctx, dictName, words)
	if err != nil {
		return nil, fmt.Errorf("failed to get dictionary entries: %w", err)
	}

	// Map words to their dictionary entries for quick lookup
	wordMap := make(map[string][]dictionary.DictionaryEntry)
	for _, entries := range dictEntries {
		if len(entries) > 0 {
			wordMap[entries[0].Word] = entries
		}
	}

	// Combine excerpts with their word meanings
	var es []ExcerptWithWords
	for _, e := range excerpts {
		ew := ExcerptWithWords{
			Excerpt: e,
			Words:   make(map[string][]dictionary.DictionaryEntry),
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
		es = append(es, ew)
	}

	sort.Slice(es, func(i, j int) bool {
		p1 := es[i].Path
		p2 := es[j].Path
		for k := 0; k < len(p1) && k < len(p2); k++ {
			if p1[k] != p2[k] {
				return p1[k] < p2[k]
			}
		}
		return len(p1) < len(p2)
	})

	// Combine excerpt with its scripture information
	scriptureName := paths[0].Scripture
	scri := s.scriptureMap[scriptureName]
	return &ExcerptTemplateData{
		Excerpts:  es,
		Scripture: scri,
	}, nil
}

// Search returns upto 100 Excerpts which match the search according to search parameters.
func (s *ExcerptService) Search(ctx context.Context, search SearchParams) (*ExcerptSearchData, error) {
	iastQuery, err := s.transliterator.Convert(search.Q, common.Transliteration(search.Tl), common.TlIAST)
	if err != nil {
		slog.Warn("transliteration failed for scripture search", "query", search.Q, "err", err)
		iastQuery = search.Q
	}
	search.Q = iastQuery

	// If no scriptures are specified, search in all of them.
	excerpts, err := s.store.Search(ctx, search.Scriptures, search)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}
	return &ExcerptSearchData{Excerpts: excerpts, Search: search}, nil
}

func NewScriptureService(index bleve.Index, conf *config.DheeConfig, transliterator *transliteration.Transliterator) *ExcerptService {
	scriptureMap := map[string]config.ScriptureDefn{}
	for _, scri := range conf.Scriptures {
		scriptureMap[scri.Name] = scri
	}

	return &ExcerptService{
		ds:             dictionary.NewBleveDictStore(index, conf),
		store:          NewBleveExcerptStore(index, conf),
		conf:           conf,
		transliterator: transliterator,
		scriptureMap:   scriptureMap,
	}
}
