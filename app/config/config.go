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
	Name                 string                `json:"name"`
	ReadableName         string                `json:"readable_name"`
	Description          string                `json:"description"`
	Attribution          string                `json:"attribution"`
	Hierarchy            []string              `json:"hierarchy"`
	Auxiliaries          []AuxiliaryDefinition `json:"auxiliaries"`
	TranslationAuxiliary string                `json:"translation_auxiliary,omitempty"`
	DataFile             string                `json:"data_file"`
	NotesFile            string                `json:"notes_file,omitempty"`
	NotesBy              string                `json:"notes_by,omitempty"`
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
	Hostnames      []string        `json:"hostnames"`
}

type ServerRuntimeConfig struct {
	Addr               string
	Port               int
	CertDir            string
	AcmeEnabled        bool
	RateLimit          int // 0 for inifinite
	BehindLoadBalancer bool
	GzipLevel          int // 0 to disable
}

func (c *DheeConfig) GetScriptureByName(name string) *ScriptureDefn {
	for _, s := range c.Scriptures {
		if s.Name == name {
			return &s
		}
	}
	return nil
}

func (c *DheeConfig) GetDictByName(name string) *DictDefn {
	for _, d := range c.Dictionaries {
		if d.Name == name {
			return &d
		}
	}
	return nil
}

func (c *DheeConfig) DictNameToId(dictName string) int {
	for idx, d := range c.Dictionaries {
		if d.Name == dictName {
			return idx
		}
	}
	return -1
}

func (c *DheeConfig) ScriptureNameToId(scriptureName string) int {
	for idx, d := range c.Scriptures {
		if d.Name == scriptureName {
			return idx
		}
	}
	return -1
}

func (c *DheeConfig) GetAuxiliaryAttributions(scriptureName string) map[string]string {
	s := c.GetScriptureByName(scriptureName)
	if s == nil {
		return nil
	}

	attributions := make(map[string]string)
	for _, aux := range s.Auxiliaries {
		if aux.Attribution != "" {
			attributions[aux.Name] = aux.Attribution
		}
	}
	return attributions
}

type AssetHashStore interface {
	FormatWithHash(path string) string
}

type PageRenderContext struct {
	Config      *DheeConfig
	AssetHashes AssetHashStore
}
