package scripture

import (
	"context"

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
	return nil, nil
}

// Search returns upto 100 Excerpts which match the search according to search parameters.
func (s *ScriptureService) Search(ctx context.Context, search SearchParams) ([]Excerpt, error) {
	return nil, nil
}

func NewScriptureService(index bleve.Index, conf config.DheeConfig) *ScriptureService {
	return &ScriptureService{
		ds:    dictionary.NewBleveDictStore(index, conf),
		store: NewBleveExcerptStore(index, conf),
		conf:  conf,
	}
}
