package dictionary

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mahesh-hegde/dhee/app/common"
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
	// _, err = s.db.Exec(`
	// 	CREATE VIRTUAL TABLE IF NOT EXISTS dhee_dictionary_spellfix USING spellfix1;
	// `)
	// if err != nil {
	// 	return fmt.Errorf("failed to create spellfix1 table: %w", err)
	// }

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

	// spellfixStmt, err := tx.Prepare("INSERT INTO dhee_dictionary_spellfix (word) VALUES (?)")
	// if err != nil {
	// return err
	// }
	// defer spellfixStmt.Close()

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

		// _, err = spellfixStmt.ExecContext(ctx, e.Word)
		// if err != nil {
		// return err
		// }
	}

	return tx.Commit()
}

func (s *SQLiteDictStore) Get(ctx context.Context, dictName string, words []string) (map[string]DictionaryEntry, error) {
	if len(words) == 0 {
		return make(map[string]DictionaryEntry), nil
	}

	ids := make([]any, len(words))
	dictId := s.conf.DictNameToId(dictName)
	for i, word := range words {
		ids[i] = fmt.Sprintf("%d:%s", dictId, word)
	}

	query := "SELECT entry FROM dhee_dictionary_entries WHERE id IN (?" + strings.Repeat(",?", len(ids)-1) + ")"

	rows, err := s.db.QueryContext(ctx, query, ids...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make(map[string]DictionaryEntry)
	for rows.Next() {
		var entryJSON []byte
		if err := rows.Scan(&entryJSON); err != nil {
			return nil, err
		}
		var entry DictionaryEntry
		if err := json.Unmarshal(entryJSON, &entry); err != nil {
			return nil, err
		}
		results[entry.Word] = entry
	}

	return results, rows.Err()
}

func (s *SQLiteDictStore) Search(ctx context.Context, dictName string, searchParams SearchParams) (SearchResults, error) {
	if searchParams.Mode == "exact" {
		query := `SELECT entry FROM dhee_dictionary_entries WHERE dict_name = ? AND word = ? ORDER BY word LIMIT 100`
		rows, err := s.db.QueryContext(ctx, query, dictName, searchParams.Query)
		if err != nil {
			return SearchResults{}, fmt.Errorf("sqlite search failed: %w", err)
		}
		defer rows.Close()

		var items []DictSearchResult
		for rows.Next() {
			var entryJSON []byte
			if err := rows.Scan(&entryJSON); err != nil {
				return SearchResults{}, err
			}
			var ent DictionaryEntry
			if err := json.Unmarshal(entryJSON, &ent); err != nil {
				return SearchResults{}, err
			}
			previews := make([]string, 0, len(ent.Meanings))
			for _, meaning := range ent.Meanings {
				previews = append(previews, meaning.Body.Plain)
			}
			items = append(items, DictSearchResult{
				IAST:     ent.IAST,
				Word:     ent.Word,
				Previews: previews,
			})
		}
		if err := rows.Err(); err != nil {
			return SearchResults{}, err
		}

		return SearchResults{Items: items, DictionaryName: dictName}, nil
	}

	var ftsQuery, ftsColumn, orderBy string
	orderBy = "ORDER BY de.word"

	switch searchParams.Mode {
	case "prefix":
		q := searchParams.Query + "*"
		ftsQuery = fmt.Sprintf("word:%s OR variants:%s", q, q)
	case "translations":
		ftsQuery = searchParams.Query
		ftsColumn = "body_text"
		orderBy = "ORDER BY de_fts.rank" // Corresponds to _score sort
	default:
		return SearchResults{}, fmt.Errorf("search mode '%s' not supported by sqlite store", searchParams.Mode)
	}

	matchClause := "de_fts.dhee_dictionary_fts MATCH ?"
	if ftsColumn != "" {
		matchClause = fmt.Sprintf("de_fts.dhee_dictionary_fts(%s) MATCH ?", ftsColumn)
	}

	query := `
		SELECT de.entry 
		FROM dhee_dictionary_fts AS de_fts JOIN dhee_dictionary_entries AS de ON de_fts.rowid = de.rowid 
		WHERE de.dict_name = ? AND ` + matchClause
	fullQuery := query + " " + orderBy + " LIMIT 100"

	rows, err := s.db.QueryContext(ctx, fullQuery, dictName, ftsQuery)
	if err != nil {
		return SearchResults{}, fmt.Errorf("sqlite search failed: %w", err)
	}
	defer rows.Close()

	var items []DictSearchResult
	for rows.Next() {
		var entryJSON []byte
		if err := rows.Scan(&entryJSON); err != nil {
			return SearchResults{}, err
		}
		var ent DictionaryEntry
		if err := json.Unmarshal(entryJSON, &ent); err != nil {
			return SearchResults{}, err
		}
		previews := make([]string, 0, len(ent.Meanings))
		for _, meaning := range ent.Meanings {
			previews = append(previews, meaning.Body.Plain)
		}
		items = append(items, DictSearchResult{
			IAST:     ent.IAST,
			Word:     ent.Word,
			Previews: previews,
		})
	}
	if err := rows.Err(); err != nil {
		return SearchResults{}, err
	}

	return SearchResults{Items: items, DictionaryName: dictName}, nil
}

func (s *SQLiteDictStore) Suggest(ctx context.Context, dictName string, p SuggestParams) (Suggestions, error) {
	query := `
		SELECT entry
		FROM dhee_dictionary_entries
		WHERE dict_name = ? AND word LIKE ?
		ORDER BY word
		LIMIT 20
	`

	rows, err := s.db.QueryContext(ctx, query, dictName, p.PartialQuery+"%")
	if err != nil {
		return Suggestions{}, fmt.Errorf("sqlite suggest failed: %w", err)
	}
	defer rows.Close()

	var items []DictSearchSuggestion
	for rows.Next() {
		var entryJSON []byte
		if err := rows.Scan(&entryJSON); err != nil {
			return Suggestions{}, err
		}
		var ent DictionaryEntry
		if err := json.Unmarshal(entryJSON, &ent); err != nil {
			return Suggestions{}, err
		}

		var preview string
		if len(ent.Meanings) > 0 {
			preview = ent.Meanings[0].Body.Plain
		}

		items = append(items, DictSearchSuggestion{
			IAST:    ent.IAST,
			Preview: preview,
		})
	}

	if err := rows.Err(); err != nil {
		return Suggestions{}, err
	}

	return Suggestions{Items: items}, nil
}

func (s *SQLiteDictStore) Related(ctx context.Context, dictName string, word string) (SearchResults, error) {
	return s.Search(ctx, dictName, SearchParams{
		Query: word + "-",
		Mode:  common.SearchMode("prefix"),
	})
}
