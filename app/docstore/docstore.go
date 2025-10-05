package docstore

import (
	"github.com/mahesh-hegde/dhee/app/dictionary"
	"github.com/mahesh-hegde/dhee/app/scripture"
)

// DocStore is a storage instrument (eg: a database) which allows efficient bulk
// insertion, retrieval and search operations

// Can all instruments provide same interface?
type DocStore interface {
	DictionaryStore() dictionary.DictStore
	ExcerptStore() scripture.ExcerptStore
}
