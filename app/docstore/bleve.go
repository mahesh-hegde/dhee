package docstore

import (
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/mahesh-hegde/dhee/app/dictionary"
	"github.com/mahesh-hegde/dhee/app/scripture"
)

type BleveDocStore struct{}

var _ mapping.Classifier = &dictionary.DictionaryEntry{}

var _ mapping.Classifier = &scripture.Excerpt{}
