package scripture

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/query"
	"github.com/mahesh-hegde/dhee/app/common"
	"github.com/mahesh-hegde/dhee/app/config"
)

type ExcerptStore interface {
	Add(ctx context.Context, scripture string, es []Excerpt) error
	Get(ctx context.Context, paths []QualifiedPath) ([]Excerpt, error)
	Search(ctx context.Context, scriptures []string, params SearchParams) ([]Excerpt, error)
}

type BleveExcerptStore struct {
	idx  bleve.Index
	conf *config.DheeConfig
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
func (b *BleveExcerptStore) Get(ctx context.Context, paths []QualifiedPath) ([]Excerpt, error) {
	ids := make([]string, len(paths))
	for i, p := range paths {
		ids[i] = fmt.Sprintf("%s:%s", p.Scripture, common.PathToString(p.Path))
	}

	if len(ids) == 0 {
		return nil, nil
	}

	query := bleve.NewDocIDQuery(ids)
	searchRequest := bleve.NewSearchRequest(query)
	searchRequest.Size = len(ids)
	searchRequest.Fields = []string{"*"}

	searchResults, err := b.idx.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	var excerpts []Excerpt
	for _, hit := range searchResults.Hits {
		e, err := docToExcerpt(hit.Fields)
		if err != nil {
			return nil, err
		}
		excerpts = append(excerpts, e)
	}
	return excerpts, nil
}

// Search implements ExcerptStore.
func (b *BleveExcerptStore) Search(ctx context.Context, scriptures []string, params SearchParams) ([]Excerpt, error) {
	var scriptureQueries []query.Query

	for _, s := range scriptures {
		q := bleve.NewTermQuery(s)
		q.SetField("scripture")
		scriptureQueries = append(scriptureQueries, q)
	}
	var scriptureQuery query.Query = bleve.NewMatchAllQuery()
	if len(scriptures) != 0 {
		scriptureQuery = bleve.NewDisjunctionQuery(scriptureQueries...)
	}

	// TODO: Support all kinds of search
	var contentQuery query.Query
	if params.Q != "" {
		sourceQuery := bleve.NewMatchQuery(params.Q)
		sourceQuery.SetField("source_text")
		romanQuery := bleve.NewMatchQuery(params.Q)
		romanQuery.SetField("roman_text")
		auxQuery := bleve.NewMatchQuery(params.Q)
		auxQuery.SetField("auxiliaries.*")
		contentQuery = bleve.NewDisjunctionQuery(sourceQuery, romanQuery)
	}

	finalQuery := bleve.NewConjunctionQuery(scriptureQuery, contentQuery)
	searchRequest := bleve.NewSearchRequest(finalQuery)
	searchRequest.Size = 100
	searchRequest.Fields = []string{"*"}

	searchResults, err := b.idx.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	var excerpts []Excerpt
	for _, hit := range searchResults.Hits {
		e, err := docToExcerpt(hit.Fields)
		if err != nil {
			slog.Info("failed to convert doc to excerpt", "err", err)
			continue
		}
		excerpts = append(excerpts, e)
	}

	return excerpts, nil
}

var _ ExcerptStore = &BleveExcerptStore{}

func NewBleveExcerptStore(idx bleve.Index, conf *config.DheeConfig) *BleveExcerptStore {
	return &BleveExcerptStore{idx: idx, conf: conf}
}

// docToExcerpt reconstructs an Excerpt from bleve fields.
// This is complex because of nested structs and slices. Bleve flattens the structure.
// A more robust way would be to store the original JSON and retrieve that.
// For now, we reconstruct it manually.
func docToExcerpt(fields map[string]any) (Excerpt, error) {
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
