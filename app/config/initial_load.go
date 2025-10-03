package config

import (
	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
)

func GetBleveIndexMappings() mapping.IndexMapping {
	panic("not implemented")
}

func LoadData(index bleve.Index, config DheeConfig) error {
	panic("Not implemented")
}

func InitDB(dataDir string, config DheeConfig) error {
	panic("Not implemented")
}
