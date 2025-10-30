package config

import (
	"github.com/mahesh-hegde/dhee/app/common"
)

type AuxiliaryDefinition struct {
	Name             string            `json:"name"`
	ReadableName     string            `json:"readable_name"`
	Language         common.Language   `json:"language"`
	Attribution      string            `json:"attribution"`
	AttributionLinks map[string]string `json:"attribution_link"`
	Note             string            `json:"note"`
}

type ScriptureDefn struct {
	Name             string                `json:"name"`
	ReadableName     string                `json:"readable_name"`
	Description      string                `json:"description"`
	Attribution      string                `json:"attribution"`
	AttributionLinks map[string]string     `json:"attribution_links"`
	Hierarchy        []string              `json:"hierarchy"`
	Auxiliaries      []AuxiliaryDefinition `json:"auxiliaries"`
	DataFile         string                `json:"data_file"`
	NotesFile        string                `json:"notes_file,omitempty"`
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
	InstanceName   string          `json:"instance_name"`
	DataDir        string          `json:"-"`
	Dictionaries   []DictDefn      `json:"dictionaries"`
	DefaultDict    string          `json:"default_dict"`
	Scriptures     []ScriptureDefn `json:"scriptures"`
	Fuzziness      int             `json:"fuzziness"`
	LogLatency     bool            `json:"log_latency"`
	TimeoutSeconds int64           `json:"timeout_seconds"`
}

func (c *DheeConfig) GetScriptureByName(name string) *ScriptureDefn {
	for _, s := range c.Scriptures {
		if s.Name == name {
			return &s
		}
	}
	return nil
}
