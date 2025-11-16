package excerpts

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/mahesh-hegde/dhee/app/common"
	"github.com/mahesh-hegde/dhee/app/config"
)

type SQLiteExcerptStore struct {
	db   *sql.DB
	conf *config.DheeConfig
}

func NewSQLiteExcerptStore(db *sql.DB, conf *config.DheeConfig) *SQLiteExcerptStore {
	return &SQLiteExcerptStore{db: db, conf: conf}
}

var _ ExcerptStore = &SQLiteExcerptStore{}

func (s *SQLiteExcerptStore) Init() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS dhee_excerpts (
			id TEXT PRIMARY KEY,
			scripture TEXT,
			sort_index TEXT,
			e BLOB
		);
		CREATE INDEX IF NOT EXISTS idx_excerpt_sort_index ON dhee_excerpts(sort_index);
	`)
	if err != nil {
		return fmt.Errorf("failed to create dhee_excerpts table: %w", err)
	}

	_, err = s.db.Exec(`
		CREATE VIRTUAL TABLE IF NOT EXISTS dhee_excerpts_fts USING fts5(
			source_t,
			roman_t,
			roman_k,
			roman_f,
			auxiliaries,
			addressees,
			notes,
			authors,
			meter,
			surfaces,
			translation
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create dhee_excerpts_fts table: %w", err)
	}
	return nil
}

func (s *SQLiteExcerptStore) Add(ctx context.Context, scripture string, es []Excerpt) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	scriptureDefn := s.conf.GetScriptureByName(scripture)
	if scriptureDefn == nil {
		return fmt.Errorf("scripture not found in config: %s", scripture)
	}

	stmt, err := tx.Prepare("INSERT INTO dhee_excerpts (id, scripture, sort_index, e) VALUES (?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	ftsStmt, err := tx.Prepare(`
		INSERT INTO dhee_excerpts_fts (
			source_t, roman_t, roman_k, roman_f, auxiliaries,
			addressees, notes, authors, meter, surfaces, translation
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer ftsStmt.Close()

	for _, e := range es {
		e.Scripture = scripture
		if e.ReadableIndex == "" {
			e.ReadableIndex = common.PathToString(e.Path)
		}
		id := fmt.Sprintf("%d:%s", s.conf.ScriptureNameToId(scripture), e.ReadableIndex)
		entryJSON, err := json.Marshal(e)
		if err != nil {
			return fmt.Errorf("failed to json encode excerpt: %w", err)
		}

		sortIndex := common.PathToSortString(e.Path)
		_, err = stmt.ExecContext(ctx, id, scripture, sortIndex, entryJSON)
		if err != nil {
			return err
		}

		sourceT := strings.Join(e.SourceText, " ")
		romanT := strings.Join(e.RomanText, " ")
		romanK := normalizeRomanTextForKwStorage(e.RomanText)
		romanF := romanK
		auxTexts := make([]string, 0, len(e.Auxiliaries))
		for _, auxObj := range e.Auxiliaries {
			auxTexts = append(auxTexts, strings.Join(auxObj.Text, " "))
		}
		var surfaces []string
		for _, glossGroup := range e.Glossings {
			for _, g := range glossGroup {
				if g.Surface != "" {
					surfaces = append(surfaces, g.Surface)
				}
			}
		}

		var translationText string
		if scriptureDefn.TranslationAuxiliary != "" {
			if aux, ok := e.Auxiliaries[scriptureDefn.TranslationAuxiliary]; ok {
				translationText = strings.Join(aux.Text, " ")
			}
		}

		_, err = ftsStmt.ExecContext(ctx,
			sourceT,
			romanT,
			romanK,
			romanF,
			strings.Join(auxTexts, " "),
			strings.Join(e.Addressees, ", "),
			strings.Join(e.Notes, " "),
			strings.Join(e.Authors, ", "),
			e.Meter,
			strings.Join(surfaces, ", "),
			translationText,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *SQLiteExcerptStore) Get(ctx context.Context, paths []QualifiedPath) ([]Excerpt, error) {
	return nil, errors.New("not implemented")
}

func (s *SQLiteExcerptStore) FindBeforeAndAfter(ctx context.Context, scripture string, idsBefore []string, idsAfter []string) (prev string, next string) {
	return "", ""
}

func (s *SQLiteExcerptStore) Search(ctx context.Context, scriptures []string, params SearchParams) ([]Excerpt, error) {
	return nil, errors.New("not implemented")
}

func (s *SQLiteExcerptStore) GetHier(ctx context.Context, scripture *config.ScriptureDefn, path []int) (*Hierarchy, error) {
	return nil, errors.New("not implemented")
}
