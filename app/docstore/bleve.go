package docstore

import (
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/mahesh-hegde/dhee/app/dictionary"
	"github.com/mahesh-hegde/dhee/app/excerpts"
)

type BleveDocStore struct{}

var _ mapping.Classifier = &dictionary.DictionaryEntryInDB{}

var _ mapping.Classifier = &excerpts.ExcerptInDB{}
