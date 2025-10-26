package dictionary

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/query"
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
		id := fmt.Sprintf("%s:%s", dictName, e.Id)
		if err := batch.Index(id, e); err != nil {
			return fmt.Errorf("failed to add item to batch: %w", err)
		}
	}

	if err := b.idx.Batch(batch); err != nil {
		return fmt.Errorf("failed to execute batch: %w", err)
	}
	return nil
}

// Get implements DictStore.
func (b *BleveDictStore) Get(ctx context.Context, dictName string, words []string) (map[string][]DictionaryEntry, error) {
	dictQuery := bleve.NewTermQuery(dictName)
	dictQuery.SetField("dict_name")

	wordQueries := make([]query.Query, len(words))
	for i, word := range words {
		q := bleve.NewTermQuery(word)
		q.SetField("word")
		wordQueries[i] = q
	}
	wordsQuery := bleve.NewDisjunctionQuery(wordQueries...)

	finalQuery := bleve.NewConjunctionQuery(dictQuery, wordsQuery)
	searchRequest := bleve.NewSearchRequest(finalQuery)
	searchRequest.Size = 100 // A reasonable limit
	searchRequest.Fields = []string{"*"}
	searchRequest.SortBy([]string{"htag", "word", "_id"})

	searchResults, err := b.idx.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("bleve search failed: %w", err)
	}

	results := make(map[string][]DictionaryEntry)
	for _, hit := range searchResults.Hits {
		word := hit.Fields["word"].(string)
		entry, err := docToDictEntry(hit.Fields)
		if err != nil {
			slog.Warn("failed to convert doc to dict entry", "err", err)
			continue
		}
		results[word] = append(results[word], entry)
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
	case "regexp":
		q := bleve.NewRegexpQuery(s.Query)
		q.SetField("word")
		vq := bleve.NewPrefixQuery(s.Query)
		vq.SetField("variants")
		wordQuery = bleve.NewDisjunctionQuery(q, vq)
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
	searchRequest.Fields = []string{"*"}
	searchRequest.SortBy([]string{"htag", "word", "_id"})

	searchResults, err := b.idx.Search(searchRequest)
	if err != nil {
		return SearchResults{}, fmt.Errorf("bleve search failed: %w", err)
	}

	var items []DictSearchResult
	for _, hit := range searchResults.Hits {
		items = append(items, DictSearchResult{
			IAST:    hit.Fields["iast"].(string),
			Word:    hit.Fields["word"].(string),
			Nagari:  hit.Fields["devanagari"].(string),
			Preview: hit.Fields["body.plain"].(string),
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
	searchRequest.Fields = []string{"iast", "devanagari", "body.plain"}

	searchResults, err := b.idx.Search(searchRequest)
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

var _ DictStore = &BleveDictStore{}

func NewBleveDictStore(idx bleve.Index, conf *config.DheeConfig) *BleveDictStore {
	return &BleveDictStore{idx: idx, conf: conf}
}

func docToDictEntry(fields map[string]interface{}) (DictionaryEntry, error) {
	var entry DictionaryEntry
	var err error

	// Helper function for safe type assertion
	getString := func(key string) string {
		if val, ok := fields[key].(string); ok {
			return val
		}
		return ""
	}

	getBool := func(key string) bool {
		if val, ok := fields[key].(bool); ok {
			return val
		}
		return false
	}

	getInt := func(key string) int {
		if val, ok := fields[key].(float64); ok { // Numbers from JSON are often float64
			return int(val)
		}
		return 0
	}

	getStringSlice := func(key string) []string {
		if val, ok := fields[key].([]interface{}); ok {
			var slice []string
			for _, item := range val {
				if str, ok := item.(string); ok {
					slice = append(slice, str)
				}
			}
			return slice
		}
		return nil
	}

	entry.DictName = getString("dict_name")
	entry.Word = getString("word")
	entry.HTag = getString("htag")
	entry.Id = getString("id")
	entry.IAST = getString("iast")
	entry.HK = getString("hk")
	entry.Devanagari = getString("devanagari")
	entry.PrintedPageNum = getString("print_page")
	entry.LexicalGender = getString("lexical_gender")
	entry.Stem = getString("stem")

	entry.Variants = getStringSlice("variants")
	entry.Cognates = getStringSlice("cognates")
	entry.LitRefs = getStringSlice("lit_refs")

	entry.HomonymNumber = getInt("homonym_number")
	entry.IsAnimalName = getBool("is_animal_name")
	entry.IsPlantName = getBool("is_plant_name")

	// Nested objects - assuming they are stored as JSON strings or flattened
	if plain, ok := fields["body.plain"].(string); ok {
		entry.Body.Plain = plain
	}
	if markup, ok := fields["body.markup"].(string); ok {
		entry.Body.Markup = markup
	}

	// For LexCat and Verb, Bleve flattens them.
	entry.LexCat.LexID = getString("lexcat.lex_id")
	entry.LexCat.Stem = getString("lexcat.stem")
	entry.LexCat.RootClass = getString("lexcat.root_class")
	entry.LexCat.IsLoan = getBool("lexcat.is_loan")
	entry.LexCat.InflictType = getString("lexcat.inflict_type")

	entry.Verb.VerbType = getString("verb.verb_type")
	entry.Verb.VerbClass = getInt("verb.verb_class")
	entry.Verb.Pada = getString("verb.pada")
	entry.Verb.Parse = getStringSlice("verb.parse")

	return entry, err
}
