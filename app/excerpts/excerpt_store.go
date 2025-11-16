package excerpts

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/query"
	"github.com/mahesh-hegde/dhee/app/common"
	"github.com/mahesh-hegde/dhee/app/config"
)

type ExcerptStore interface {
	Init() error
	Add(ctx context.Context, scripture string, es []Excerpt) error
	Get(ctx context.Context, paths []QualifiedPath) ([]Excerpt, error)
	// FindBeforeAndAfter, given a set of possible idsBefore and idsAfter in priority order,
	// finds the immediate previous and next ID with one query
	FindBeforeAndAfter(ctx context.Context, scripture string, idsBefore []string, idsAfter []string) (prev string, next string)
	Search(ctx context.Context, scriptures []string, params SearchParams) ([]Excerpt, error)
	GetHier(ctx context.Context, scripture *config.ScriptureDefn, path []int) (*Hierarchy, error)
}

type BleveExcerptStore struct {
	idx  bleve.Index
	conf *config.DheeConfig
}

// GetHier implements ExcerptStore.
func (b *BleveExcerptStore) GetHier(ctx context.Context, scripture *config.ScriptureDefn, path []int) (*Hierarchy, error) {
	// This might could be made more efficient, but why care when our whole dataset fits in memory?
	if len(path) >= len(scripture.Hierarchy) {
		return nil, fmt.Errorf("cannot obtain hierarchy for a leaf element")
	}
	qs := fmt.Sprintf("%d:%s", b.conf.ScriptureNameToId(scripture.Name), common.PathToString(path))
	if len(path) != 0 {
		qs += "."
	}
	qPref := bleve.NewPrefixQuery(qs)
	qPref.SetField("_id")

	var qFinal query.Query = qPref

	if len(path) < len(scripture.Hierarchy)-1 {
		limitVerses := qs + "[^.]*(.1)*"
		qReg := bleve.NewRegexpQuery(limitVerses)
		qReg.SetField("_id")
		qFinal = bleve.NewConjunctionQuery(qPref, qReg)
	}

	searchRequest := bleve.NewSearchRequest(qFinal)
	searchRequest.Size = 10000 // Max verses I'd expect anywhere
	searchRequest.Fields = []string{"_id"}
	searchResults, err := b.idx.SearchInContext(ctx, searchRequest)
	if err != nil {
		return nil, err
	}
	plen := len(path)
	known := make(map[int]struct{}, 0)
	childs := make([]int, 0)
	for _, hit := range searchResults.Hits {
		id := hit.ID
		_, pthStr, ok := strings.Cut(id, ":")
		if !ok {
			continue
		}
		chldPath, err := common.StringToPath(pthStr)
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

// FindBeforeAndAfter implements ExcerptStore.
func (b *BleveExcerptStore) FindBeforeAndAfter(ctx context.Context, scripture string, idsBefore []string, idsAfter []string) (prev string, next string) {
	ids := make([]string, len(idsBefore)+len(idsAfter))
	idset := make(map[string]struct{}, 0)
	for i, p := range idsBefore {
		ids[i] = fmt.Sprintf("%d:%s", b.conf.ScriptureNameToId(scripture), p)
	}
	for i, p := range idsAfter {
		ids[i+len(idsBefore)] = fmt.Sprintf("%d:%s", b.conf.ScriptureNameToId(scripture), p)
	}
	if len(ids) == 0 {
		return "", ""
	}
	query := bleve.NewDocIDQuery(ids)
	searchRequest := bleve.NewSearchRequest(query)
	searchRequest.Size = len(ids)
	searchRequest.Fields = []string{"_id"}

	searchResults, err := b.idx.SearchInContext(ctx, searchRequest)
	if err != nil {
		slog.Error("error when querying existence of IDs")
		return "", ""
	}
	for _, hit := range searchResults.Hits {
		idset[hit.ID] = struct{}{}
	}
	var before, after string
	for _, id := range idsBefore {
		fullId := fmt.Sprintf("%d:%s", b.conf.ScriptureNameToId(scripture), id)
		if _, found := idset[fullId]; found {
			before = id
			break
		}
	}
	for _, id := range idsAfter {
		fullId := fmt.Sprintf("%d:%s", b.conf.ScriptureNameToId(scripture), id)
		if _, found := idset[fullId]; found {
			after = id
			break
		}
	}
	return before, after
}

func (b *BleveExcerptStore) Init() error {
	return nil
}

// Add implements ExcerptStore.
func (b *BleveExcerptStore) Add(ctx context.Context, scripture string, es []Excerpt) error {
	batch := b.idx.NewBatch()
	for _, e := range es {
		e.Scripture = scripture
		if e.ReadableIndex == "" {
			e.ReadableIndex = common.PathToString(e.Path)
		}
		dbEntry := prepareExcerptForDb(&e)
		id := fmt.Sprintf("%d:%s", b.conf.ScriptureNameToId(scripture), e.ReadableIndex)
		err := batch.Index(id, &dbEntry)
		if err != nil {
			return err
		}
	}
	return b.idx.Batch(batch)
}

// Get implements ExcerptStore.
func (b *BleveExcerptStore) Get(ctx context.Context, paths []QualifiedPath) ([]Excerpt, error) {
	ids := make([]string, len(paths))
	for i, p := range paths {
		ids[i] = fmt.Sprintf("%d:%s", b.conf.ScriptureNameToId(p.Scripture), common.PathToString(p.Path))
	}

	if len(ids) == 0 {
		return nil, nil
	}

	query := bleve.NewDocIDQuery(ids)
	searchRequest := bleve.NewSearchRequest(query)
	searchRequest.Size = len(ids)
	searchRequest.Fields = []string{"*"}

	searchResults, err := b.idx.SearchInContext(ctx, searchRequest)
	if err != nil {
		return nil, err
	}

	var excerpts []Excerpt
	for _, hit := range searchResults.Hits {
		e, err := docToExcerpt(hit.Fields)
		if err != nil {
			return nil, err
		}
		excerpts = append(excerpts, e)
	}
	return excerpts, nil
}

// Search implements ExcerptStore.
func (b *BleveExcerptStore) Search(ctx context.Context, scriptures []string, params SearchParams) ([]Excerpt, error) {
	var scriptureQueries []query.Query

	for _, s := range scriptures {
		q := bleve.NewTermQuery(s)
		q.SetField("scripture")
		scriptureQueries = append(scriptureQueries, q)
	}
	var scriptureQuery query.Query = bleve.NewMatchAllQuery()
	if len(scriptures) != 0 {
		scriptureQuery = bleve.NewDisjunctionQuery(scriptureQueries...)
	}

	var queryMaker func(string, string) query.Query
	switch params.Mode {
	case common.SearchRegex:
		queryMaker = func(q string, field string) query.Query {
			bq := bleve.NewRegexpQuery(q)
			if field == "roman_t" {
				// redirect to roman_k which contains whole text (not analyzed).
				field = "roman_k"
			}
			bq.SetField(field)
			return bq
		}
	case common.SearchASCII:
		queryMaker = func(q string, field string) query.Query {
			bq := bleve.NewMatchQuery(q)
			if field == "roman_t" {
				// redirect to roman_folded which contains ascii folded text
				field = "roman_f"
			}
			bq.SetField(field)
			return bq
		}
	case common.SearchFuzzy:
		queryMaker = func(q string, field string) query.Query {
			bq := bleve.NewFuzzyQuery(q)
			bq.SetField(field)
			bq.Fuzziness = b.conf.Fuzziness
			return bq
		}
	case common.SearchPrefix:
		queryMaker = func(q string, field string) query.Query {
			bq := bleve.NewPrefixQuery(q)
			bq.SetField(field)
			return bq
		}
	case common.SearchTranslations:
		queryMaker = func(q string, field string) query.Query {
			// ignore field that is specified
			// TODO: only pick English auxiliaries?
			bqs := make([]query.Query, 0)
			for _, scripture := range scriptures {
				bq := bleve.NewMatchQuery(q)
				sc := b.conf.GetScriptureByName(scripture)
				bq.SetField("auxiliaries." + sc.TranslationAuxiliary)
				bqs = append(bqs, bq)
			}
			return bleve.NewDisjunctionQuery(bqs...)
		}
	default:
		queryMaker = func(q string, field string) query.Query {
			bq := bleve.NewMatchQuery(q)
			bq.SetField(field)
			return bq
		}
	}
	var contentQuery query.Query
	if params.Q != "" {
		romanQuery := queryMaker(params.Q, "roman_t")
		contentQuery = romanQuery
	}

	finalQuery := bleve.NewConjunctionQuery(scriptureQuery, contentQuery)
	searchRequest := bleve.NewSearchRequest(finalQuery)
	searchRequest.Size = 100
	searchRequest.Fields = []string{"*"}
	if params.Mode == common.SearchRegex {
		searchRequest.SortBy([]string{"sort_index"})
	} else {
		searchRequest.SortBy([]string{"-_score", "sort_index"})
	}

	searchResults, err := b.idx.SearchInContext(ctx, searchRequest)
	if err != nil {
		return nil, err
	}

	var excerpts []Excerpt
	for _, hit := range searchResults.Hits {
		e, err := docToExcerpt(hit.Fields)
		if err != nil {
			slog.Info("failed to convert doc to excerpt", "err", err)
			continue
		}
		excerpts = append(excerpts, e)
	}

	return excerpts, nil
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

func prepareExcerptForDb(e *Excerpt) ExcerptInDB {
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

	return ExcerptInDB{
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

var _ ExcerptStore = &BleveExcerptStore{}

func NewBleveExcerptStore(idx bleve.Index, conf *config.DheeConfig) *BleveExcerptStore {
	return &BleveExcerptStore{idx: idx, conf: conf}
}

func docToExcerpt(fields map[string]any) (Excerpt, error) {
	raw, ok := fields["e"].(string)
	if !ok {
		return Excerpt{}, fmt.Errorf("missing field e in document")
	}
	var e Excerpt
	err := json.Unmarshal([]byte(raw), &e)
	return e, err
}
