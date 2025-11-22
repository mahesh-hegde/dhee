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
		dbEntry := prepareExcerptForDb(b.conf, &e)
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
		e, err := bleveDocToExcerpt(hit.Fields)
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
			re := q
			if !strings.HasPrefix(re, "^") && !strings.HasPrefix(re, ".*") {
				re = ".*" + re
			}
			if !strings.HasSuffix(re, "$") && !strings.HasSuffix(re, ".*") {
				re = re + ".*"
			}
			bq := bleve.NewRegexpQuery(re)
			if field == "roman_t" {
				// redirect to roman_k which contains whole text (not analyzed).
				field = "roman_k"
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
	case common.SearchExact:
		queryMaker = func(q string, field string) query.Query {
			bq := bleve.NewMatchQuery(q)
			bq.SetField(field)
			bq.SetBoost(1.2)
			bqa := bleve.NewMatchQuery(q)
			bqa.SetField("roman_f")
			bqa.SetBoost(0.9)
			return bleve.NewDisjunctionQuery(bq, bqa)
		}
	default:
		return nil, common.NewUserVisibleError(422, "Unsupported search mode")
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
	searchRequest.Highlight = bleve.NewHighlightWithStyle("html")

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

		e, err := bleveDocToExcerpt(hit.Fields)
		if err != nil {
			slog.Info("failed to convert doc to excerpt", "err", err)
			continue
		}

		// If search mode is translations, try to get highlighted fragments from Bleve and override the auxiliary
		if params.Mode == common.SearchTranslations {
			// Bleve returns fragments in hit.Fragments map[string][]string
			if frags, ok := hit.Fragments["translation"]; ok && len(frags) > 0 {
				hl := strings.Join(frags, " ... ")
				if sc := b.conf.GetScriptureByName(e.Scripture); sc != nil && sc.TranslationAuxiliary != "" {
					if e.Auxiliaries == nil {
						e.Auxiliaries = make(map[string]Auxiliary)
					}
					e.Auxiliaries[sc.TranslationAuxiliary] = Auxiliary{Text: []string{hl}}
				}
			}
		}
		excerpts = append(excerpts, e)
	}
	return excerpts, nil
}

var _ ExcerptStore = &BleveExcerptStore{}

func NewBleveExcerptStore(idx bleve.Index, conf *config.DheeConfig) *BleveExcerptStore {
	return &BleveExcerptStore{idx: idx, conf: conf}
}

func bleveDocToExcerpt(fields map[string]any) (Excerpt, error) {
	raw, ok := fields["e"].(string)
	if !ok {
		return Excerpt{}, fmt.Errorf("missing field e in document")
	}
	var e Excerpt
	err := json.Unmarshal([]byte(raw), &e)
	return e, err
}
