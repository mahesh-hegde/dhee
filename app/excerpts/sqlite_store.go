package excerpts

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
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
	if len(paths) == 0 {
		return nil, nil
	}

	ids := make([]any, len(paths))
	for i, p := range paths {
		ids[i] = fmt.Sprintf("%d:%s", s.conf.ScriptureNameToId(p.Scripture), common.PathToString(p.Path))
	}

	query := "SELECT e FROM dhee_excerpts WHERE id IN (?" + strings.Repeat(",?", len(ids)-1) + ")"

	rows, err := s.db.QueryContext(ctx, query, ids...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var excerpts []Excerpt
	for rows.Next() {
		var excerptJSON []byte
		if err := rows.Scan(&excerptJSON); err != nil {
			return nil, err
		}
		var excerpt Excerpt
		if err := json.Unmarshal(excerptJSON, &excerpt); err != nil {
			return nil, err
		}
		excerpts = append(excerpts, excerpt)
	}

	return excerpts, rows.Err()
}

func (s *SQLiteExcerptStore) FindBeforeAndAfter(ctx context.Context, scripture string, idsBefore []string, idsAfter []string) (prev string, next string) {
	allIds := make([]any, 0, len(idsBefore)+len(idsAfter))

	scriptureId := s.conf.ScriptureNameToId(scripture)
	for _, p := range idsBefore {
		id := fmt.Sprintf("%d:%s", scriptureId, p)
		allIds = append(allIds, id)
	}
	for _, p := range idsAfter {
		id := fmt.Sprintf("%d:%s", scriptureId, p)
		allIds = append(allIds, id)
	}

	if len(allIds) == 0 {
		return "", ""
	}

	query := "SELECT id FROM dhee_excerpts WHERE id IN (?" + strings.Repeat(",?", len(allIds)-1) + ")"
	rows, err := s.db.QueryContext(ctx, query, allIds...)
	if err != nil {
		return "", ""
	}
	defer rows.Close()

	idset := make(map[string]struct{})
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return "", ""
		}
		idset[id] = struct{}{}
	}

	var before, after string
	for _, id := range idsBefore {
		fullId := fmt.Sprintf("%d:%s", scriptureId, id)
		if _, found := idset[fullId]; found {
			before = id
			break
		}
	}
	for _, id := range idsAfter {
		fullId := fmt.Sprintf("%d:%s", scriptureId, id)
		if _, found := idset[fullId]; found {
			after = id
			break
		}
	}
	return before, after
}

func (s *SQLiteExcerptStore) Search(ctx context.Context, scriptures []string, params SearchParams) ([]Excerpt, error) {
	var ftsQuery string
	orderBy := "ORDER BY T1.rank, T2.sort_index"

	q := params.Q
	switch params.Mode {
	case common.SearchASCII:
		ftsQuery = "roman_f:" + q
	case common.SearchPrefix:
		ftsQuery = "roman_t:" + q + "*"
	case common.SearchTranslations:
		ftsQuery = "translation:" + q
	case common.SearchFuzzy:
		return nil, errors.New("fuzzy search is not supported on excerpts with sqlite store")
	case common.SearchRegex:
		return nil, errors.New("regex search is not supported on excerpts with sqlite store")
	default: // "exact" from controller, which means match query in bleve. The bleve code defaults to a match query.
		ftsQuery = "roman_t:" + q
	}

	scripturePlaceholders := "?"
	if len(scriptures) > 1 {
		scripturePlaceholders += strings.Repeat(",?", len(scriptures)-1)
	}

	query := `
		SELECT T2.e
		FROM dhee_excerpts_fts AS T1 JOIN dhee_excerpts AS T2 ON T1.rowid = T2.rowid
		WHERE T2.scripture IN (` + scripturePlaceholders + `) AND T1.dhee_excerpts_fts MATCH ?
	`

	fullQuery := query + " " + orderBy + " LIMIT 100"

	args := make([]any, 0, len(scriptures)+1)
	for _, s := range scriptures {
		args = append(args, s)
	}
	args = append(args, ftsQuery)

	rows, err := s.db.QueryContext(ctx, fullQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("sqlite search failed: %w", err)
	}
	defer rows.Close()

	var excerpts []Excerpt
	for rows.Next() {
		var excerptJSON []byte
		if err := rows.Scan(&excerptJSON); err != nil {
			return nil, err
		}
		var excerpt Excerpt
		if err := json.Unmarshal(excerptJSON, &excerpt); err != nil {
			return nil, err
		}
		excerpts = append(excerpts, excerpt)
	}

	return excerpts, rows.Err()
}

func (s *SQLiteExcerptStore) GetHier(ctx context.Context, scripture *config.ScriptureDefn, path []int) (*Hierarchy, error) {
	if len(path) >= len(scripture.Hierarchy) {
		return nil, fmt.Errorf("cannot obtain hierarchy for a leaf element")
	}

	sortPrefix := common.PathToSortString(path)
	if len(path) > 0 {
		sortPrefix += "."
	}

	query := "SELECT sort_index FROM dhee_excerpts WHERE scripture = ? AND sort_index LIKE ?"
	rows, err := s.db.QueryContext(ctx, query, scripture.Name, sortPrefix+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	plen := len(path)
	known := make(map[int]struct{})
	var childs []int
	for rows.Next() {
		var sortIndex string
		if err := rows.Scan(&sortIndex); err != nil {
			return nil, err
		}
		chldPath, err := common.StringToPath(sortIndex)
		if err != nil {
			continue
		}
		if len(chldPath) <= plen {
			continue
		}
		chld := chldPath[plen]
		if _, seen := known[chld]; seen {
			continue
		}
		known[chld] = struct{}{}
		childs = append(childs, chld)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	lineage := make([]HierParent, 0)
	for level, num := range path {
		hierParent := HierParent{
			Type:     scripture.Hierarchy[level],
			Number:   num,
			FullPath: common.PathToString(path[:level+1]),
		}
		lineage = append(lineage, hierParent)
	}
	sort.Ints(childs)
	return &Hierarchy{
		Scripture: scripture,
		Path:      lineage,
		ChildType: scripture.Hierarchy[plen],
		Children:  childs,
		IsLeaf:    len(lineage)+1 == len(scripture.Hierarchy),
	}, nil
}
