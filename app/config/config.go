package config

import (
	"github.com/mahesh-hegde/dhee/app/dictionary"
	"github.com/mahesh-hegde/dhee/app/scripture"
)

type DheeConfig struct {
	InstanceName string
	Dictionaries []dictionary.DictDefn
	Scriptures   []scripture.ScriptureDefn
}

type DheeApplicationContext struct {
	Conf  DheeConfig
	Store DocStore
}
