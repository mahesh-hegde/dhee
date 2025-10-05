package scripture

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/query"
	"github.com/mahesh-hegde/dhee/app/config"
)

type ExcerptStore interface {
	Add(ctx context.Context, scripture string, es []Excerpt) error
	Get(ctx context.Context, paths []QualifiedPath) []Excerpt
	Search(ctx context.Context, scriptures []string, params SearchParams) []Excerpt
}

type BleveExcerptStore struct {
	idx  bleve.Index
	conf config.DheeConfig
}

// Add implements ExcerptStore.
func (b *BleveExcerptStore) Add(ctx context.Context, scripture string, es []Excerpt) error {
	batch := b.idx.NewBatch()
	for _, e := range es {
		e.Scripture = scripture
		id := fmt.Sprintf("%s:%s", scripture, e.ReadableIndex)
		err := batch.Index(id, e)
		if err != nil {
			return err
		}
	}
	return b.idx.Batch(batch)
}

// Get implements ExcerptStore.
func (b *BleveExcerptStore) Get(ctx context.Context, paths []QualifiedPath) []Excerpt {
	ids := make([]string, len(paths))
	for i, p := range paths {
		pathTokens := make([]string, len(p.Path))
		for j, token := range p.Path {
			pathTokens[j] = fmt.Sprintf("%d", token)
		}
		// NOTE: This assumes ReadableIndex is the same as the path joined by dots.
		// This is true for rigveda, but might not be for others.
		// A more robust solution would be to query by scripture and path fields.
		ids[i] = fmt.Sprintf("%s:%s", p.Scripture, strings.Join(pathTokens, "."))
	}

	if len(ids) == 0 {
		return nil
	}

	query := bleve.NewDocIDQuery(ids)
	searchRequest := bleve.NewSearchRequest(query)
	searchRequest.Size = len(ids)
	searchRequest.Fields = []string{"*"}

	searchResults, err := b.idx.Search(searchRequest)
	if err != nil {
		log.Printf("bleve search failed: %v", err)
		return nil
	}

	var excerpts []Excerpt
	for _, hit := range searchResults.Hits {
		e, err := docToExcerpt(hit.Fields)
		if err != nil {
			log.Printf("failed to convert doc to excerpt: %v", err)
			continue
		}
		excerpts = append(excerpts, e)
	}

	return excerpts
}

// Search implements ExcerptStore.
func (b *BleveExcerptStore) Search(ctx context.Context, scriptures []string, params SearchParams) []Excerpt {
	var scriptureQueries []query.Query
	for _, s := range scriptures {
		q := bleve.NewTermQuery(s)
		q.SetField("scripture")
		scriptureQueries = append(scriptureQueries, q)
	}
	scriptureQuery := bleve.NewDisjunctionQuery(scriptureQueries...)

	var contentQuery query.Query
	if params.Q != "" {
		sourceQuery := bleve.NewMatchQuery(params.Q)
		sourceQuery.SetField("source_text")
		romanQuery := bleve.NewMatchQuery(params.Q)
		romanQuery.SetField("roman_text")
		// TODO: Make search fields configurable
		contentQuery = bleve.NewDisjunctionQuery(sourceQuery, romanQuery)
	}

	finalQuery := bleve.NewConjunctionQuery(scriptureQuery, contentQuery)
	searchRequest := bleve.NewSearchRequest(finalQuery)
	searchRequest.Size = 100
	searchRequest.Fields = []string{"*"}

	searchResults, err := b.idx.Search(searchRequest)
	if err != nil {
		log.Printf("bleve search failed: %v", err)
		return nil
	}

	var excerpts []Excerpt
	for _, hit := range searchResults.Hits {
		e, err := docToExcerpt(hit.Fields)
		if err != nil {
			log.Printf("failed to convert doc to excerpt: %v", err)
			continue
		}
		excerpts = append(excerpts, e)
	}

	return excerpts
}

var _ ExcerptStore = &BleveExcerptStore{}

func NewBleveExcerptStore(idx bleve.Index, conf config.DheeConfig) *BleveExcerptStore {
	return &BleveExcerptStore{idx: idx, conf: conf}
}

// docToExcerpt reconstructs an Excerpt from bleve fields.
// This is complex because of nested structs and slices. Bleve flattens the structure.
// A more robust way would be to store the original JSON and retrieve that.
// For now, we reconstruct it manually.
func docToExcerpt(fields map[string]interface{}) (Excerpt, error) {
	// A trick to avoid manual mapping: convert map to JSON bytes, then unmarshal.
	// This works if the field names in the map match the JSON tags in the struct.
	bytes, err := json.Marshal(fields)
	if err != nil {
		return Excerpt{}, fmt.Errorf("failed to marshal fields to json: %w", err)
	}

	var e Excerpt
	if err := json.Unmarshal(bytes, &e); err != nil {
		return Excerpt{}, fmt.Errorf("failed to unmarshal json to excerpt: %w", err)
	}

	return e, nil
}
