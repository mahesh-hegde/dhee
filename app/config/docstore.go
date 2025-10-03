package config

import (
	"github.com/mahesh-hegde/dhee/app/dictionary"
	"github.com/mahesh-hegde/dhee/app/scripture"
)

// DocStore is a storage instrument (eg: a database) which allows efficient bulk
// insertion, retrieval and search operations

// Can all instruments provide same interface?

type DictionaryStore interface {
	Add(dictionary string, es []dictionary.DictionaryEntry) error
	Update(dictionary string, es dictionary.DictionaryEntry) error
	Del(dictionary string, es dictionary.DictionaryEntry) error
	Get(dictionary string, words []string) (map[string]dictionary.DictionaryEntry, error)
	GetByIds(dictionary string, ids []string) (map[string]dictionary.DictionaryEntry, error)
	Search(dictionary string, s dictionary.SearchParams) dictionary.SearchResults
	Suggest(dictionary string, s dictionary.SuggestParams) dictionary.Suggestions
}

type ExcerptStore interface {
	Add(scripture string, es []scripture.Excerpt) error
	Update(es scripture.Excerpt) error
	Del(es scripture.Excerpt) error
	Get(paths []scripture.QualifiedPath)
}

type DocStore interface {
	DictionaryStore() DictionaryStore
	ExcerptStore() ExcerptStore
}
