package excerpts

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/gob"
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
			surfaces
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

	stmt, err := tx.Prepare("INSERT INTO dhee_excerpts (id, scripture, sort_index, e) VALUES (?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	ftsStmt, err := tx.Prepare(`
		INSERT INTO dhee_excerpts_fts (
			source_t, roman_t, roman_k, roman_f, auxiliaries,
			addressees, notes, authors, meter, surfaces
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
		var buf bytes.Buffer
		if err := gob.NewEncoder(&buf).Encode(e); err != nil {
			return fmt.Errorf("failed to gob encode excerpt: %w", err)
		}

		sortIndex := common.PathToSortString(e.Path)
		_, err = stmt.ExecContext(ctx, id, scripture, sortIndex, buf.Bytes())
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
