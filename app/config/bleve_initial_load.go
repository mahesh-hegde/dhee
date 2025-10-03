package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"

	"github.com/mahesh-hegde/dhee/app/dictionary"
	"github.com/mahesh-hegde/dhee/app/scripture"

	"github.com/blevesearch/bleve/v2/analysis/analyzer/custom"
	"github.com/blevesearch/bleve/v2/analysis/token/lowercase"
	"github.com/blevesearch/bleve/v2/analysis/tokenizer/whitespace"
)

func GetBleveIndexMappings() mapping.IndexMapping {
	// Base index mapping
	indexMapping := mapping.NewIndexMapping()

	// Add custom analyzer for Sanskrit/IAST/Devanagari
	err := indexMapping.AddCustomAnalyzer("sanskrit_ws",
		map[string]any{
			"type":      custom.Name,
			"tokenizer": whitespace.Name, // unicode.Name,
			"token_filters": []string{
				lowercase.Name,
				// "asciifolding", // GPT suggested but this doesn't exists
			},
		})
	if err != nil {
		slog.Error("error when defining index", "err", err)
		os.Exit(1)
	}

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

func pathToString(path []int) string {
	var parts []string
	for _, p := range path {
		parts = append(parts, strconv.Itoa(p))
	}
	return strings.Join(parts, ".")
}

const batchSize = 128

// loadJSONL is a generic helper to load data from a JSONL file into a bleve index.
func loadJSONL[T any](index bleve.Index, dataFile string, idFunc func(T) string, enrichFunc func(*T)) error {
	file, err := os.Open(dataFile)
	if err != nil {
		return fmt.Errorf("failed to open data file %s: %w", dataFile, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	batch := index.NewBatch()
	count := 0

	for scanner.Scan() {
		var entry T
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			slog.Warn("failed to unmarshal line", "err", err)
			continue
		}

		if enrichFunc != nil {
			enrichFunc(&entry)
		}

		id := idFunc(entry)
		if err := batch.Index(id, entry); err != nil {
			return fmt.Errorf("failed to add item to batch: %w", err)
		}
		count++

		if count >= batchSize {
			if err := index.Batch(batch); err != nil {
				return fmt.Errorf("failed to execute batch: %w", err)
			}
			batch = index.NewBatch()
			count = 0
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %w", err)
	}

	// Execute the final batch
	if count > 0 {
		if err := index.Batch(batch); err != nil {
			return fmt.Errorf("failed to execute final batch: %w", err)
		}
	}

	slog.Info("Successfully loaded data", "file", dataFile)
	return nil
}

func LoadData(index bleve.Index, dataDir string, config *DheeConfig) error {
	// Load scriptures
	for _, sc := range config.Scriptures {
		slog.Info("Loading scripture", "name", sc.Name)
		err := loadJSONL(index, path.Join(dataDir, sc.DataFile),
			func(e scripture.Excerpt) string {
				return fmt.Sprintf("%s:%s", sc.Name, pathToString(e.Path))
			},
			func(e *scripture.Excerpt) {
				e.Scripture = sc.Name
			},
		)
		if err != nil {
			return fmt.Errorf("failed to load scripture %s: %w", sc.Name, err)
		}
	}

	// Load dictionaries
	for _, dict := range config.Dictionaries {
		slog.Info("Loading dictionary", "name", dict.Name)
		err := loadJSONL(index, path.Join(dataDir, dict.DataFile),
			func(e dictionary.DictionaryEntry) string {
				return fmt.Sprintf("%s:%s", dict.Name, e.Id)
			},
			func(e *dictionary.DictionaryEntry) {
				e.DictName = dict.Name
			},
		)
		if err != nil {
			return fmt.Errorf("failed to load dictionary %s: %w", dict.Name, err)
		}
	}

	return nil
}

func InitDB(dataDir string, config *DheeConfig) (bleve.Index, error) {
	dbPath := filepath.Join(dataDir, "docstore.bleve")
	index, err := bleve.Open(dbPath)

	if err == bleve.ErrorIndexPathDoesNotExist {
		slog.Info("Creating new bleve index", "path", dbPath)
		mapping := GetBleveIndexMappings()
		index, err = bleve.New(dbPath, mapping)
		if err != nil {
			return nil, fmt.Errorf("failed to create new bleve index: %w", err)
		}

		slog.Info("Loading initial data into the index...")
		if err := LoadData(index, dataDir, config); err != nil {
			// Cleanup created index on load failure
			os.RemoveAll(dbPath)
			return nil, fmt.Errorf("failed to load data: %w", err)
		}
		slog.Info("Initial data loaded successfully.")
	} else if err != nil {
		return nil, fmt.Errorf("failed to open bleve index: %w", err)
	} else {
		slog.Info("Opened existing bleve index", "path", dbPath)
	}

	return index, nil
}
