package dictionary

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/query"
	"github.com/mahesh-hegde/dhee/app/config"
)

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

type BleveDictStore struct {
	idx  bleve.Index
	conf *config.DheeConfig
}

// Add implements DictStore.
func (b *BleveDictStore) Add(ctx context.Context, dictName string, es []DictionaryEntry) error {
	batch := b.idx.NewBatch()
	for _, e := range es {
		e.DictName = dictName
		dbEntry := prepareDictEntryForDb(&e)
		id := fmt.Sprintf("%d:%s", b.conf.DictNameToId(dictName), e.Word)
		if err := batch.Index(id, &dbEntry); err != nil {
			return fmt.Errorf("failed to add item to batch: %w", err)
		}
	}

	if err := b.idx.Batch(batch); err != nil {
		return fmt.Errorf("failed to execute batch: %w", err)
	}
	return nil
}

// Get implements DictStore.
func (b *BleveDictStore) Get(ctx context.Context, dictName string, words []string) (map[string]DictionaryEntry, error) {
	dictQuery := bleve.NewTermQuery(dictName)
	dictQuery.SetField("dict_name")

	wordQueries := make([]string, len(words))
	for i, word := range words {
		wordQueries[i] = fmt.Sprintf("%d:%s", b.conf.DictNameToId(dictName), word)
	}
	wordsQuery := bleve.NewDocIDQuery(wordQueries)

	finalQuery := bleve.NewConjunctionQuery(dictQuery, wordsQuery)
	searchRequest := bleve.NewSearchRequest(finalQuery)

	searchRequest.Size = len(wordQueries) // A reasonable limit
	searchRequest.Fields = []string{"word", "_id", "e"}
	searchRequest.SortBy([]string{"_id"})

	searchResults, err := b.idx.SearchInContext(ctx, searchRequest)
	if err != nil {
		return nil, fmt.Errorf("bleve search failed: %w", err)
	}

	results := make(map[string]DictionaryEntry)
	for _, hit := range searchResults.Hits {
		word := hit.Fields["word"].(string)
		entry, err := docToDictEntry(hit.Fields)
		if err != nil {
			slog.Warn("failed to convert doc to dict entry", "err", err)
			continue
		}
		results[word] = entry
	}

	return results, nil
}

// Related implements DictStore.
func (b *BleveDictStore) Related(ctx context.Context, dictName string, word string) (SearchResults, error) {
	return b.Search(ctx, dictName, SearchParams{
		Query: word + "-",
		Mode:  "prefix",
	})
}

// Search implements DictStore.
func (b *BleveDictStore) Search(ctx context.Context, dictName string, s SearchParams) (SearchResults, error) {
	dictQuery := bleve.NewTermQuery(dictName)
	dictQuery.SetField("dict_name")

	var wordQuery query.Query

	switch s.Mode {
	case "prefix":
		q := bleve.NewPrefixQuery(s.Query)
		q.SetField("word")
		vq := bleve.NewPrefixQuery(s.Query)
		vq.SetField("variants")
		wordQuery = bleve.NewDisjunctionQuery(q, vq)
	case "fuzzy":
		q := bleve.NewFuzzyQuery(s.Query)
		q.SetField("word")
		q.Fuzziness = b.conf.Fuzziness
		vq := bleve.NewFuzzyQuery(s.Query)
		vq.SetField("variants")
		vq.Fuzziness = b.conf.Fuzziness
		wordQuery = bleve.NewDisjunctionQuery(q, vq)
	case "regex":
		q := bleve.NewRegexpQuery(s.Query)
		q.SetField("word")
		vq := bleve.NewPrefixQuery(s.Query)
		vq.SetField("variants")
		wordQuery = bleve.NewDisjunctionQuery(q, vq)
	case "translations":
		q := bleve.NewMatchQuery(s.Query)
		q.SetField("body_text")
		wordQuery = q
	default: // exact
		q := bleve.NewTermQuery(s.Query)
		q.SetField("word")
		vq := bleve.NewPrefixQuery(s.Query)
		vq.SetField("variants")
		wordQuery = bleve.NewDisjunctionQuery(q, vq)
	}

	finalQuery := bleve.NewConjunctionQuery(dictQuery, wordQuery)
	searchRequest := bleve.NewSearchRequest(finalQuery)
	searchRequest.Size = 100
	searchRequest.Fields = []string{"_id", "e"}
	if s.Mode == "fuzzy" || s.Mode == "translations" {
		// TODO: add htag
		searchRequest.SortBy([]string{"-_score", "_id"})
	} else {
		// TODO: add htag
		searchRequest.SortBy([]string{"_id"})
	}

	searchResults, err := b.idx.SearchInContext(ctx, searchRequest)
	if err != nil {
		return SearchResults{}, fmt.Errorf("bleve search failed: %w", err)
	}

	var items []DictSearchResult
	for _, hit := range searchResults.Hits {
		ent, err := docToDictEntry(hit.Fields)
		if err != nil {
			return SearchResults{}, err
		}
		previews := make([]string, 0, len(ent.Meanings))
		for _, meaning := range ent.Meanings {
			previews = append(previews, meaning.Body.Plain)
		}
		items = append(items, DictSearchResult{
			IAST:     ent.IAST,
			Word:     ent.Word,
			Previews: previews,
		})
	}

	return SearchResults{Items: items, DictionaryName: dictName}, nil
}

// Suggest implements DictStore.
func (b *BleveDictStore) Suggest(ctx context.Context, dictName string, s SuggestParams) (Suggestions, error) {
	dictQuery := bleve.NewTermQuery(dictName)
	dictQuery.SetField("dict_name")

	wordQuery := bleve.NewPrefixQuery(s.PartialQuery)
	wordQuery.SetField("word")

	finalQuery := bleve.NewConjunctionQuery(dictQuery, wordQuery)
	searchRequest := bleve.NewSearchRequest(finalQuery)
	searchRequest.Size = 20 // Limit suggestions
	searchRequest.Fields = []string{"iast", "body.plain"}

	searchResults, err := b.idx.SearchInContext(ctx, searchRequest)
	if err != nil {
		return Suggestions{}, fmt.Errorf("bleve search failed: %w", err)
	}

	var items []DictSearchSuggestion
	for _, hit := range searchResults.Hits {
		items = append(items, DictSearchSuggestion{
			IAST:    hit.Fields["iast"].(string),
			Nagari:  hit.Fields["devanagari"].(string),
			Preview: hit.Fields["body.plain"].(string),
		})
	}

	return Suggestions{Items: items}, nil
}

func (b *BleveDictStore) Init() error {
	return nil
}

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

var _ DictStore = &BleveDictStore{}

func NewBleveDictStore(idx bleve.Index, conf *config.DheeConfig) *BleveDictStore {
	return &BleveDictStore{idx: idx, conf: conf}
}

func docToDictEntry(fields map[string]any) (DictionaryEntry, error) {
	raw, ok := fields["e"].(string)
	if !ok {
		return DictionaryEntry{}, fmt.Errorf("missing field e in document")
	}
	var d DictionaryEntry
	err := json.Unmarshal([]byte(raw), &d)
	return d, err
}
