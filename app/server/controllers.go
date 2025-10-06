package server

import (
	"github.com/blevesearch/bleve/v2"
	"github.com/mahesh-hegde/dhee/app/config"
	"github.com/mahesh-hegde/dhee/app/dictionary"
	"github.com/mahesh-hegde/dhee/app/scripture"
)

type DheeController struct {
	ds *dictionary.DictionaryService
	es *scripture.ExcerptService
}

func NewDheeController(index bleve.Index, conf *config.DheeConfig) *DheeController {
	return &DheeController{
		ds: dictionary.NewDictionaryService(index, conf),
		es: scripture.NewScriptureService(index, conf),
	}
}
