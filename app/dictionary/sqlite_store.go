package dictionary

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/mahesh-hegde/dhee/app/config"
)

type SQLiteDictStore struct {
	db   *sql.DB
	conf *config.DheeConfig
}

func NewSQLiteDictStore(db *sql.DB, conf *config.DheeConfig) *SQLiteDictStore {
	return &SQLiteDictStore{db: db, conf: conf}
}

var _ DictStore = &SQLiteDictStore{}

func (s *SQLiteDictStore) Init() error {
	// Main table
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS dhee_dictionary_entries (
			id TEXT PRIMARY KEY,
			dict_name TEXT,
			word TEXT,
			entry BLOB
		);
		CREATE INDEX IF NOT EXISTS idx_dict_word ON dhee_dictionary_entries(word);
	`)
	if err != nil {
		return fmt.Errorf("failed to create dhee_dictionary_entries table: %w", err)
	}

	// FTS table
	_, err = s.db.Exec(`
		CREATE VIRTUAL TABLE IF NOT EXISTS dhee_dictionary_fts USING fts5(
			word,
			variants,
			lit_refs,
			body_text
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create dhee_dictionary_fts table: %w", err)
	}

	// Spellfix table
	_, err = s.db.Exec(`
		CREATE VIRTUAL TABLE IF NOT EXISTS dhee_dictionary_spellfix USING spellfix1;
	`)
	if err != nil {
		return fmt.Errorf("failed to create spellfix1 table: %w", err)
	}

	return nil
}

func (s *SQLiteDictStore) Add(ctx context.Context, dictName string, es []DictionaryEntry) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT INTO dhee_dictionary_entries (id, dict_name, word, entry) VALUES (?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	ftsStmt, err := tx.Prepare("INSERT INTO dhee_dictionary_fts (word, variants, lit_refs, body_text) VALUES (?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer ftsStmt.Close()

	spellfixStmt, err := tx.Prepare("INSERT INTO dhee_dictionary_spellfix (word) VALUES (?)")
	if err != nil {
		return err
	}
	defer spellfixStmt.Close()

	for _, e := range es {
		e.DictName = dictName
		id := fmt.Sprintf("%d:%s", s.conf.DictNameToId(dictName), e.Word)

		entryJSON, err := json.Marshal(e)
		if err != nil {
			return fmt.Errorf("failed to json encode dictionary entry: %w", err)
		}

		_, err = stmt.ExecContext(ctx, id, dictName, e.Word, entryJSON)
		if err != nil {
			return err
		}

		bodyText := []string{}
		for _, meaning := range e.Meanings {
			bodyText = append(bodyText, meaning.Body.Plain)
		}
		variants := getAllVariants(&e)
		litRefs := getAllLitRefs(&e)

		_, err = ftsStmt.ExecContext(ctx,
			e.Word,
			strings.Join(variants, ", "),
			strings.Join(litRefs, ", "),
			strings.Join(bodyText, " "),
		)
		if err != nil {
			return err
		}

		_, err = spellfixStmt.ExecContext(ctx, e.Word)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *SQLiteDictStore) Get(ctx context.Context, dictName string, words []string) (map[string]DictionaryEntry, error) {
	return nil, errors.New("not implemented")
}

func (s *SQLiteDictStore) Search(ctx context.Context, dictName string, searchParams SearchParams) (SearchResults, error) {
	return SearchResults{}, errors.New("not implemented")
}

func (s *SQLiteDictStore) Suggest(ctx context.Context, dictName string, p SuggestParams) (Suggestions, error) {
	return Suggestions{}, errors.New("not implemented")
}

func (s *SQLiteDictStore) Related(ctx context.Context, dictName string, word string) (SearchResults, error) {
	return SearchResults{}, errors.New("not implemented")
}
