package docstore

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis"
	"github.com/blevesearch/bleve/v2/analysis/char/asciifolding"
	"github.com/blevesearch/bleve/v2/mapping"

	"github.com/mahesh-hegde/dhee/app/common"
	"github.com/mahesh-hegde/dhee/app/config"
	"github.com/mahesh-hegde/dhee/app/dictionary"
	"github.com/mahesh-hegde/dhee/app/excerpts"

	"github.com/blevesearch/bleve/v2/analysis/analyzer/custom"
	"github.com/blevesearch/bleve/v2/analysis/token/lowercase"
	"github.com/blevesearch/bleve/v2/analysis/tokenizer/unicode"
	"github.com/blevesearch/bleve/v2/registry"
	"github.com/yuin/goldmark"
)

// Possible auxiliaries from all texts we know
// TODO: loop over config
var possibleAuxiliaries = []string{"griffith", "oldenberg", "pada"}

// --- accent folding char filter ---
type AccentFoldingCharFilter struct{}

func (ac AccentFoldingCharFilter) Filter(input []byte) []byte {
	replacer := strings.NewReplacer(common.FoldableAccentsList...)
	replaced := replacer.Replace(string(input))
	return []byte(replaced)
}

func init() {
	registry.RegisterCharFilter("accent_fold", func(config map[string]interface{}, cache *registry.Cache) (analysis.CharFilter, error) {
		return AccentFoldingCharFilter{}, nil
	})
}

func parseNotesFile(filePath string) (map[string]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("opening notes file: %w", err)
	}
	defer file.Close()

	notes := make(map[string]string)
	scanner := bufio.NewScanner(file)
	var currentID string
	var currentContent strings.Builder

	md := goldmark.New()

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "## ") {
			if currentID != "" && currentContent.Len() > 0 {
				var buf bytes.Buffer
				if err := md.Convert([]byte(currentContent.String()), &buf); err != nil {
					slog.Warn("failed to convert markdown to html", "id", currentID, "err", err)
				} else {
					notes[currentID] = buf.String()
				}
			}
			currentID = strings.TrimSpace(strings.TrimPrefix(line, "## "))
			currentContent.Reset()
		} else {
			if currentID != "" {
				currentContent.WriteString(line)
				currentContent.WriteString("\n")
			}
		}
	}
	if currentID != "" && currentContent.Len() > 0 {
		var buf bytes.Buffer
		if err := md.Convert([]byte(currentContent.String()), &buf); err != nil {
			slog.Warn("failed to convert markdown to html", "id", currentID, "err", err)
		} else {
			notes[currentID] = buf.String()
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning notes file: %w", err)
	}

	return notes, nil
}

// --- bleve mappings ---
func GetBleveIndexMappings() mapping.IndexMapping {
	// Base index mapping
	indexMapping := mapping.NewIndexMapping()

	// Add custom analyzer for Sanskrit/IAST/Devanagari
	err1 := indexMapping.AddCustomAnalyzer("sanskrit_ws",
		map[string]any{
			"type":         custom.Name,
			"char_filters": []string{"accent_fold"},
			"tokenizer":    unicode.Name,
			"token_filters": []string{
				lowercase.Name,
			},
		})
	err2 := indexMapping.AddCustomAnalyzer("ascii_folding", map[string]any{
		"type":         custom.Name,
		"char_filters": []string{asciifolding.Name},
		"tokenizer":    unicode.Name,
		"token_filters": []string{
			lowercase.Name,
		},
	})
	if err1 != nil || err2 != nil {
		slog.Error("error when defining index", "err1", err1, "err2", err2)
		os.Exit(1)
	}

	// ----- ExcerptInDB mapping -----
	excerptMapping := mapping.NewDocumentMapping()
	eField := mapping.NewKeywordFieldMapping()
	eField.Store = true
	eField.Index = false
	excerptMapping.AddFieldMappingsAt("e", eField) // stored only

	excerptMapping.AddFieldMappingsAt("scripture", mapping.NewKeywordFieldMapping())

	sField := mapping.NewTextFieldMapping()
	sField.Analyzer = "sanskrit_ws"
	excerptMapping.AddFieldMappingsAt("source_t", sField)

	rField := mapping.NewTextFieldMapping()
	rField.Analyzer = "sanskrit_ws"
	excerptMapping.AddFieldMappingsAt("roman_t", rField)

	// -- roman text as keyword for regex search --
	rkField := mapping.NewKeywordFieldMapping()
	excerptMapping.AddFieldMappingsAt("roman_k", rkField)

	// -- roman text ascii folded --
	rfField := mapping.NewTextFieldMapping()
	rfField.Analyzer = "ascii_folding"
	excerptMapping.AddFieldMappingsAt("roman_f", rfField)

	excerptMapping.AddFieldMappingsAt("view_index", mapping.NewKeywordFieldMapping())
	excerptMapping.AddFieldMappingsAt("sort_index", mapping.NewKeywordFieldMapping())

	// Authors
	excerptMapping.AddFieldMappingsAt("authors", mapping.NewTextFieldMapping())
	// Addressees
	excerptMapping.AddFieldMappingsAt("addressees", mapping.NewTextFieldMapping())
	// Group
	excerptMapping.AddFieldMappingsAt("group", mapping.NewTextFieldMapping())

	// Meter
	excerptMapping.AddFieldMappingsAt("meter", mapping.NewKeywordFieldMapping())

	// Notes
	excerptMapping.AddFieldMappingsAt("notes", mapping.NewTextFieldMapping())
	excerptMapping.AddFieldMappingsAt("surfaces", mapping.NewKeywordFieldMapping())

	auxMap := mapping.NewDocumentMapping()
	for _, ax := range possibleAuxiliaries {
		auxMap.AddFieldMappingsAt(ax, mapping.NewTextFieldMapping())
	}
	excerptMapping.AddSubDocumentMapping("auxiliaries", auxMap)

	// ----- DictionaryEntryInDB mapping -----
	dictMapping := mapping.NewDocumentMapping()

	dictMapping.AddFieldMappingsAt("dict_name", mapping.NewKeywordFieldMapping())
	dictMapping.AddFieldMappingsAt("word", mapping.NewKeywordFieldMapping())

	ef := mapping.NewKeywordFieldMapping()
	ef.Store, ef.Index = true, false
	dictMapping.AddFieldMappingsAt("e", ef)

	dictMapping.AddFieldMappingsAt("sid", mapping.NewKeywordFieldMapping())
	dictMapping.AddFieldMappingsAt("variants", mapping.NewKeywordFieldMapping())
	dictMapping.AddFieldMappingsAt("lit_refs", mapping.NewTextFieldMapping())

	bodyField := mapping.NewTextFieldMapping()
	dictMapping.AddFieldMappingsAt("body_text", bodyField)

	indexMapping.AddDocumentMapping("excerpt", excerptMapping)
	indexMapping.AddDocumentMapping("dictionary_entry", dictMapping)
	indexMapping.TypeField = "_type"

	return indexMapping
}

// --- data loading ---
const batchSize = 1024

func loadDictionaryData(store dictionary.DictStore, dict config.DictDefn, dataDir string) error {
	dataFile := path.Join(dataDir, dict.DataFile)
	file, err := os.Open(dataFile)
	if err != nil {
		return fmt.Errorf("failed to open data file %s: %w", dataFile, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	entries := make([]dictionary.DictionaryEntry, 0, batchSize)

	for scanner.Scan() {
		var entry dictionary.DictionaryEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			slog.Warn("failed to unmarshal line", "err", err)
			continue
		}

		entries = append(entries, entry)

		if len(entries) >= batchSize {
			slog.Info("ingesting dictionary batch", "size", len(entries))
			if err := store.Add(context.Background(), dict.Name, entries); err != nil {
				return fmt.Errorf("failed to execute batch: %w", err)
			}
			entries = make([]dictionary.DictionaryEntry, 0, batchSize)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %w", err)
	}

	// Execute the final batch
	if len(entries) > 0 {
		slog.Info("executing final batch", "size", len(entries))
		if err := store.Add(context.Background(), dict.Name, entries); err != nil {
			return fmt.Errorf("failed to execute final batch: %w", err)
		}
	}

	slog.Info("Successfully loaded data", "file", dataFile)
	return nil
}

func loadExcerptsData(store excerpts.ExcerptStore, sc config.ScriptureDefn, dataDir string) error {
	slog.Info("Loading scripture", "name", sc.Name)

	var notes map[string]string
	if sc.NotesFile != "" {
		var err error
		notes, err = parseNotesFile(path.Join(dataDir, sc.NotesFile))
		if err != nil {
			return fmt.Errorf("failed to parse notes for %s: %w", sc.Name, err)
		}
		slog.Info("loaded notes", "count", len(notes), "scripture", sc.Name)
	}

	dataFile := path.Join(dataDir, sc.DataFile)
	file, err := os.Open(dataFile)
	if err != nil {
		return fmt.Errorf("failed to open data file %s: %w", dataFile, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	entries := make([]excerpts.Excerpt, 0, batchSize)

	for scanner.Scan() {
		var entry excerpts.Excerpt
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			slog.Warn("failed to unmarshal line", "err", err)
			continue
		}

		if notes != nil {
			if note, ok := notes[entry.ReadableIndex]; ok {
				entry.Notes = []string{note}
			}
		}
		entries = append(entries, entry)

		if len(entries) >= batchSize {
			slog.Info("ingesting excerpts batch", "size", len(entries))
			if err := store.Add(context.Background(), sc.Name, entries); err != nil {
				return fmt.Errorf("failed to execute batch: %w", err)
			}
			entries = make([]excerpts.Excerpt, 0, batchSize)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %w", err)
	}
	if len(entries) > 0 {
		slog.Info("executing final batch", "size", len(entries))
		if err := store.Add(context.Background(), sc.Name, entries); err != nil {
			return fmt.Errorf("failed to execute final batch: %w", err)
		}
	}
	return nil
}

// --- LoadData with conversions ---
func LoadInitialData(dictStore dictionary.DictStore, excerptStore excerpts.ExcerptStore, dataDir string, config *config.DheeConfig) error {
	if err := dictStore.Init(); err != nil {
		return fmt.Errorf("failed to init dict store: %w", err)
	}
	if err := excerptStore.Init(); err != nil {
		return fmt.Errorf("failed to init excerpt store: %w", err)
	}
	// Load scriptures
	for _, sc := range config.Scriptures {
		if err := loadExcerptsData(excerptStore, sc, dataDir); err != nil {
			return fmt.Errorf("failed to load scripture %s: %w", sc.Name, err)
		}
	}

	// Load dictionaries
	for _, dict := range config.Dictionaries {
		slog.Info("Loading dictionary", "name", dict.Name)
		if err := loadDictionaryData(dictStore, dict, dataDir); err != nil {
			return fmt.Errorf("failed to load dictionary %s: %w", dict.Name, err)
		}
	}
	return nil
}

func InitDB(store, dataDir string, config *config.DheeConfig) (io.Closer, error) {
	if store == "bleve" {
		dbPath := filepath.Join(dataDir, "docstore.bleve")

		_, err := os.Stat(dbPath)

		if errors.Is(err, os.ErrNotExist) {
			slog.Info("Creating new bleve index", "path", dbPath)
			mapping := GetBleveIndexMappings()
			index, err := bleve.New(dbPath, mapping)
			if err != nil {
				return nil, fmt.Errorf("failed to create new bleve index: %w", err)
			}

			slog.Info("Loading initial data into the index...")
			dictStore := dictionary.NewBleveDictStore(index, config)
			excerptStore := excerpts.NewBleveExcerptStore(index, config)

			if err := LoadInitialData(dictStore, excerptStore, dataDir, config); err != nil {
				// Cleanup created index on load failure
				os.RemoveAll(dbPath)
				return nil, fmt.Errorf("failed to load data: %w", err)
			}
			slog.Info("Initial data loaded successfully.")
			if err := index.Close(); err != nil {
				return nil, err
			}
		} else if err != nil {
			return nil, fmt.Errorf("failed to open bleve index: %w", err)
		}
		index, err := bleve.OpenUsing(dbPath, map[string]any{"read_only": true})
		if err != nil {
			return nil, err
		}
		slog.Info("Opened existing bleve index", "path", dbPath)
		return index, nil
	} else if store == "sqlite" {
		dbPath := path.Join(dataDir, "dhee.db")
		_, err := os.Stat(dbPath)
		if errors.Is(err, os.ErrNotExist) {
			db, err := NewSQLiteDB(dataDir, false)
			if err != nil {
				return nil, fmt.Errorf("error creating sqlite db: %w", err)
			}

			dictStore := dictionary.NewSQLiteDictStore(db, config)
			excerptStore := excerpts.NewSQLiteExcerptStore(db, config)
			err = LoadInitialData(dictStore, excerptStore, dataDir, config)
			if err != nil {
				db.Close()
				os.Remove(dbPath)
				return nil, fmt.Errorf("error loading initial data into sqlite: %w", err)
			}

			slog.Info("Optimizing FTS indexes")
			if _, err := db.Exec("INSERT INTO dhee_dictionary_fts(dhee_dictionary_fts) VALUES('optimize')"); err != nil {
				slog.Warn("failed to optimize dictionary fts", "err", err)
			}
			if _, err := db.Exec("INSERT INTO dhee_excerpts_fts(dhee_excerpts_fts) VALUES('optimize')"); err != nil {
				slog.Warn("failed to optimize excerpts fts", "err", err)
			}
			return db, nil
		} else if err != nil {
			return nil, fmt.Errorf("error checking sqlite db: %w", err)
		}
		return nil, nil // Already exists
	}
	return nil, fmt.Errorf("unknown store: %s", store)
}
