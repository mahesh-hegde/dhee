package scripture

import (
	"context"

	"github.com/blevesearch/bleve/v2"
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
	panic("unimplemented")
}

// Get implements ExcerptStore.
func (b *BleveExcerptStore) Get(ctx context.Context, paths []QualifiedPath) []Excerpt {
	panic("unimplemented")
}

// Search implements ExcerptStore.
func (b *BleveExcerptStore) Search(ctx context.Context, scriptures []string, params SearchParams) []Excerpt {
	panic("unimplemented")
}

var _ ExcerptStore = &BleveExcerptStore{}

func NewBleveExcerptStore(idx bleve.Index, conf config.DheeConfig) *BleveExcerptStore {
	return &BleveExcerptStore{idx: idx, conf: conf}
}
