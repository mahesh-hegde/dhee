package config

import (
	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
)

func GetBleveIndexMappings() mapping.IndexMapping {
	// Base index mapping
	indexMapping := mapping.NewIndexMapping()

	// Add custom analyzer for Sanskrit/IAST/Devanagari
	indexMapping.AddCustomAnalyzer("sanskrit_ws",
		map[string]any{
			"type":      "custom",
			"tokenizer": "whitespace",
			"token_filters": []string{
				"to_lower",
				"asciifolding", // keeps search robust across diacritics
			},
		})

	// ----- Excerpt mapping -----
	excerptMapping := mapping.NewDocumentMapping()

	// ReadableIndex
	excerptMapping.AddFieldMappingsAt("readable_index", mapping.NewKeywordFieldMapping())

	// Path
	excerptMapping.AddFieldMappingsAt("path", mapping.NewNumericFieldMapping())

	// Source & roman text (use sanskrit_ws analyzer)
	sourceField := mapping.NewTextFieldMapping()
	sourceField.Analyzer = "sanskrit_ws"
	excerptMapping.AddFieldMappingsAt("source_text", sourceField)

	romanField := mapping.NewTextFieldMapping()
	romanField.Analyzer = "sanskrit_ws"
	excerptMapping.AddFieldMappingsAt("roman_text", romanField)

	// Authors
	excerptMapping.AddFieldMappingsAt("authors", mapping.NewTextFieldMapping())

	// Meter
	excerptMapping.AddFieldMappingsAt("meter", mapping.NewKeywordFieldMapping())

	// Notes
	excerptMapping.AddFieldMappingsAt("notes", mapping.NewTextFieldMapping())

	// Links
	linkMapping := mapping.NewDocumentMapping()
	linkMapping.AddFieldMappingsAt("url", mapping.NewKeywordFieldMapping())
	linkMapping.AddFieldMappingsAt("name", mapping.NewTextFieldMapping())
	excerptMapping.AddSubDocumentMapping("links", linkMapping)

	// Glossings
	glossingMapping := mapping.NewDocumentMapping()
	glossingMapping.AddFieldMappingsAt("surface", mapping.NewTextFieldMapping())
	glossingMapping.AddFieldMappingsAt("lemma", mapping.NewTextFieldMapping())
	glossingMapping.AddFieldMappingsAt("gramm", mapping.NewKeywordFieldMapping())
	glossingMapping.AddFieldMappingsAt("case", mapping.NewKeywordFieldMapping())
	glossingMapping.AddFieldMappingsAt("number", mapping.NewKeywordFieldMapping())
	glossingMapping.AddFieldMappingsAt("gender", mapping.NewKeywordFieldMapping())
	glossingMapping.AddFieldMappingsAt("tense", mapping.NewKeywordFieldMapping())
	glossingMapping.AddFieldMappingsAt("voice", mapping.NewKeywordFieldMapping())
	glossingMapping.AddFieldMappingsAt("person", mapping.NewKeywordFieldMapping())
	glossingMapping.AddFieldMappingsAt("mood", mapping.NewKeywordFieldMapping())
	glossingMapping.AddFieldMappingsAt("root", mapping.NewTextFieldMapping())
	glossingMapping.AddFieldMappingsAt("modifiers", mapping.NewKeywordFieldMapping())
	glossingMapping.AddFieldMappingsAt("constituents", mapping.NewTextFieldMapping())
	excerptMapping.AddSubDocumentMapping("glossings", glossingMapping)

	// Auxiliaries
	auxMapping := func(analyzer string) *mapping.DocumentMapping {
		m := mapping.NewDocumentMapping()
		m.AddFieldMappingsAt("path", mapping.NewNumericFieldMapping())
		textField := mapping.NewTextFieldMapping()
		textField.Analyzer = analyzer
		m.AddFieldMappingsAt("text", textField)
		return m
	}
	excerptMapping.AddSubDocumentMapping("auxiliaries.griffith", auxMapping("sanskrit_ws"))
	excerptMapping.AddSubDocumentMapping("auxiliaries.pada", auxMapping("sanskrit_ws"))
	excerptMapping.AddSubDocumentMapping("auxiliaries.oldenberg", auxMapping("sanskrit_ws"))

	// ----- DictionaryEntry mapping -----
	dictMapping := mapping.NewDocumentMapping()

	wordField := mapping.NewTextFieldMapping()
	wordField.Analyzer = "sanskrit_ws"
	dictMapping.AddFieldMappingsAt("word", wordField)

	dictMapping.AddFieldMappingsAt("htag", mapping.NewKeywordFieldMapping())
	dictMapping.AddFieldMappingsAt("id", mapping.NewKeywordFieldMapping())

	iastField := mapping.NewTextFieldMapping()
	iastField.Analyzer = "sanskrit_ws"
	dictMapping.AddFieldMappingsAt("iast", iastField)

	devField := mapping.NewTextFieldMapping()
	devField.Analyzer = "sanskrit_ws"
	dictMapping.AddFieldMappingsAt("devanagari", devField)

	varField := mapping.NewTextFieldMapping()
	varField.Analyzer = "sanskrit_ws"
	dictMapping.AddFieldMappingsAt("variants", varField)

	dictMapping.AddFieldMappingsAt("print_page", mapping.NewKeywordFieldMapping())
	dictMapping.AddFieldMappingsAt("cognates", mapping.NewTextFieldMapping())
	dictMapping.AddFieldMappingsAt("lit_refs", mapping.NewTextFieldMapping())
	dictMapping.AddFieldMappingsAt("lexical_gender", mapping.NewKeywordFieldMapping())

	stemField := mapping.NewTextFieldMapping()
	stemField.Analyzer = "sanskrit_ws"
	dictMapping.AddFieldMappingsAt("stem", stemField)

	dictMapping.AddFieldMappingsAt("is_animal_name", mapping.NewBooleanFieldMapping())
	dictMapping.AddFieldMappingsAt("is_plant_name", mapping.NewBooleanFieldMapping())

	// Body
	bodyMapping := mapping.NewDocumentMapping()
	bodyField := mapping.NewTextFieldMapping()
	bodyField.Analyzer = "sanskrit_ws"
	bodyMapping.AddFieldMappingsAt("plain", bodyField)
	dictMapping.AddSubDocumentMapping("body", bodyMapping)

	// LexCat
	lexcatMapping := mapping.NewDocumentMapping()
	lexcatMapping.AddFieldMappingsAt("lex_id", mapping.NewKeywordFieldMapping())
	lexcatMapping.AddFieldMappingsAt("stem", stemField)
	lexcatMapping.AddFieldMappingsAt("root_class", mapping.NewKeywordFieldMapping())
	lexcatMapping.AddFieldMappingsAt("is_loan", mapping.NewBooleanFieldMapping())
	lexcatMapping.AddFieldMappingsAt("inflict_type", mapping.NewKeywordFieldMapping())
	dictMapping.AddSubDocumentMapping("lexcat", lexcatMapping)

	// Verb
	verbMapping := mapping.NewDocumentMapping()
	verbMapping.AddFieldMappingsAt("verb_type", mapping.NewKeywordFieldMapping())
	verbMapping.AddFieldMappingsAt("verb_class", mapping.NewNumericFieldMapping())
	verbMapping.AddFieldMappingsAt("pada", mapping.NewKeywordFieldMapping())
	verbMapping.AddFieldMappingsAt("parse", mapping.NewTextFieldMapping())
	dictMapping.AddSubDocumentMapping("verb", verbMapping)

	// Register types
	indexMapping.AddDocumentMapping("excerpt", excerptMapping)
	indexMapping.AddDocumentMapping("dictionary_entry", dictMapping)

	// Defaults
	indexMapping.DefaultMapping = excerptMapping
	indexMapping.TypeField = "_type"

	return indexMapping
}

func LoadData(index bleve.Index, config DheeConfig) error {
	panic("Not implemented")
}

func InitDB(dataDir string, config DheeConfig) error {
	panic("Not implemented")
}
