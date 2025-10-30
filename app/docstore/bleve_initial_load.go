package docstore

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
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
	"github.com/mahesh-hegde/dhee/app/scripture"

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

var replacer = strings.NewReplacer(common.FoldableAccentsList...)

func normalizeRomanTextForKwStorage(txt []string) string {
	var result []string
	for _, t := range txt {
		// Why would we do this?
		// short vowels in the dataset have accented chars which do not match while searching.
		result = append(result, replacer.Replace(t))
	}
	return strings.Join(result, " ")
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

func prepareDictEntryForDb(e *dictionary.DictionaryEntry) dictionary.DictionaryEntryInDB {
	// Marshal full entry
	entryJSON, err := json.Marshal(e)
	if err != nil {
		slog.Error("unexpected error", "err", err)
		panic(err)
	}

	bodyText := e.Body.Plain
	return dictionary.DictionaryEntryInDB{
		DictName:       e.DictName,
		Word:           e.Word,
		SId:            e.Id,
		Entry:          string(entryJSON),
		Variants:       e.Variants,
		LitRefs:        e.LitRefs,
		PrintedPageNum: e.PrintedPageNum,
		BodyText:       bodyText,
	}
}

func prepareExcerptForDb(e *scripture.Excerpt) scripture.ExcerptInDB {
	entryJSON, err := json.Marshal(e)
	if err != nil {
		slog.Error("unexpected error", "err", err)
		panic(err)
	}

	aux := make(map[string]string)
	for name, auxObj := range e.Auxiliaries {
		aux[name] = strings.Join(auxObj.Text, " ")
	}

	var surfaces []string
	for _, glossGroup := range e.Glossings {
		for _, g := range glossGroup {
			if g.Surface != "" {
				surfaces = append(surfaces, g.Surface)
			}
		}
	}

	return scripture.ExcerptInDB{
		E:           string(entryJSON),
		Scripture:   e.Scripture,
		SourceT:     strings.Join(e.SourceText, " "),
		RomanT:      strings.Join(e.RomanText, " "),
		RomanK:      normalizeRomanTextForKwStorage(e.RomanText),
		RomanF:      normalizeRomanTextForKwStorage(e.RomanText),
		ViewIndex:   common.PathToString(e.Path),
		SortIndex:   common.PathToSortString(e.Path),
		Auxiliaries: aux,
		Addressees:  e.Addressees,
		Notes:       strings.Join(e.Notes, " "),
		Authors:     e.Authors,
		Meter:       e.Meter,
		Surfaces:    surfaces,
	}
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
	dictMapping.AddFieldMappingsAt("print_page", mapping.NewKeywordFieldMapping())

	bodyField := mapping.NewTextFieldMapping()
	dictMapping.AddFieldMappingsAt("body_text", bodyField)

	indexMapping.AddDocumentMapping("excerpt", excerptMapping)
	indexMapping.AddDocumentMapping("dictionary_entry", dictMapping)
	indexMapping.TypeField = "_type"

	return indexMapping
}

// --- data loading ---
const batchSize = 1024

func loadJSONL[T any, R any](index bleve.Index, dataFile string, idFunc func(T) string, enrichFunc func(T) R) error {
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

		id := idFunc(entry)
		enriched := enrichFunc(entry)

		if err := batch.Index(id, enriched); err != nil {
			return fmt.Errorf("failed to add item to batch: %w", err)
		}
		count++

		if count >= batchSize {
			slog.Info("ingest batch", "size", count)
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

// --- LoadData with conversions ---
func LoadData(index bleve.Index, dataDir string, config *config.DheeConfig) error {
	// Load scriptures
	for _, sc := range config.Scriptures {
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

		err := loadJSONL(index, path.Join(dataDir, sc.DataFile),
			func(e *scripture.Excerpt) string {
				return fmt.Sprintf("%s:%s", sc.Name, common.PathToString(e.Path))
			},
			func(e *scripture.Excerpt) *scripture.ExcerptInDB {
				e.Scripture = sc.Name
				if notes != nil {
					if note, ok := notes[e.ReadableIndex]; ok {
						e.Notes = []string{note}
					}
				}
				dbObj := prepareExcerptForDb(e)
				return &dbObj
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
			func(e *dictionary.DictionaryEntry) string {
				return fmt.Sprintf("%s:%s:%s", dict.Name, e.Word, e.Id)
			},
			func(e *dictionary.DictionaryEntry) *dictionary.DictionaryEntryInDB {
				e.DictName = dict.Name
				dbObj := prepareDictEntryForDb(e)
				return &dbObj
			},
		)
		if err != nil {
			return fmt.Errorf("failed to load dictionary %s: %w", dict.Name, err)
		}
	}
	return nil
}

func InitDB(dataDir string, config *config.DheeConfig) (bleve.Index, error) {
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
