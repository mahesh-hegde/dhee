package docstore

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"
	"strings"

	"github.com/mahesh-hegde/dhee/app/common"
	"github.com/mahesh-hegde/dhee/app/config"
	"github.com/mahesh-hegde/dhee/app/dictionary"
	"github.com/mahesh-hegde/dhee/app/excerpts"
	"github.com/mahesh-hegde/dhee/app/transliteration"
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

func parseNotesFile(filePath string, mc *MarkdownConverter, scripture config.ScriptureDefn) (map[string]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("opening notes file: %w", err)
	}
	defer file.Close()

	notes := make(map[string]string)
	scanner := bufio.NewScanner(file)
	var currentID string
	var currentContent strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "## ") {
			if currentID != "" && currentContent.Len() > 0 {
				html, err := mc.ConvertToHTML(currentContent.String(), scripture)
				if err != nil {
					slog.Warn("failed to convert markdown to html", "id", currentID, "err", err)
				} else {
					notes[currentID] = html
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
		html, err := mc.ConvertToHTML(currentContent.String(), scripture)
		if err != nil {
			slog.Warn("failed to convert markdown to html", "id", currentID, "err", err)
		} else {
			notes[currentID] = html
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning notes file: %w", err)
	}

	return notes, nil
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

func loadExcerptsData(store excerpts.ExcerptStore, sc config.ScriptureDefn, dataDir string, mc *MarkdownConverter) error {
	slog.Info("Loading scripture", "name", sc.Name)

	var notes map[string]string
	if sc.NotesFile != "" {
		var err error
		notes, err = parseNotesFile(path.Join(dataDir, sc.NotesFile), mc, sc)
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

	transliterator, err := transliteration.NewTransliterator(transliteration.TlOptions{})
	if err != nil {
		return fmt.Errorf("failed to create transliterator: %w", err)
	}
	mc := NewMarkdownConverter(dictStore, transliterator, config)

	// Load dictionaries
	for _, dict := range config.Dictionaries {
		slog.Info("Loading dictionary", "name", dict.Name)
		if err := loadDictionaryData(dictStore, dict, dataDir); err != nil {
			return fmt.Errorf("failed to load dictionary %s: %w", dict.Name, err)
		}
	}

	// Load scriptures
	for _, sc := range config.Scriptures {
		if err := loadExcerptsData(excerptStore, sc, dataDir, mc); err != nil {
			return fmt.Errorf("failed to load scripture %s: %w", sc.Name, err)
		}
	}

	return nil
}

func InitDB(store, dataDir string, config *config.DheeConfig) (io.Closer, error) {
	switch store {
	case "sqlite":
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
