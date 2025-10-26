package config

import (
	"github.com/mahesh-hegde/dhee/app/common"
)

type AuxiliaryDefinition struct {
	Name         string          `json:"name"`
	ReadableName string          `json:"readable_name"`
	Language     common.Language `json:"language"`
}

type ScriptureDefn struct {
	Name         string                `json:"name"`
	ReadableName string                `json:"readable_name"`
	Description  string                `json:"description"`
	Collector    string                `json:"collector"`
	Hierarchy    []string              `json:"hierarchy"`
	Auxiliaries  []AuxiliaryDefinition `json:"auxiliaries"`
	DataFile     string                `json:"data_file"`
}
type DictDefn struct {
	// A name slug used in URLs. Eg: monier-williams
	Name string `json:"name"`
	// A human readable name used in titles. Eg: "Monier-Williams Sanskrit-English dictionary"
	ReadableName   string                 `json:"readable_name"`
	SourceLanguage common.Language        `json:"source_language"`
	TargetLanguage common.Language        `json:"target_language"`
	WordEncoding   common.Transliteration `json:"word_encoding"`

	// File with entries encoded as JSONL
	DataFile string `json:"data_file"`
}

type DheeConfig struct {
	InstanceName string          `json:"instance_name"`
	DataDir      string          `json:"-"`
	Dictionaries []DictDefn      `json:"dictionaries"`
	DefaultDict  string          `json:"default_dict"`
	Scriptures   []ScriptureDefn `json:"scriptures"`
	Fuzziness    int             `json:"fuzziness"`
}
