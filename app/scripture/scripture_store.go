package scripture

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/query"
	"github.com/mahesh-hegde/dhee/app/common"
	"github.com/mahesh-hegde/dhee/app/config"
)

type ExcerptStore interface {
	Add(ctx context.Context, scripture string, es []Excerpt) error
	Get(ctx context.Context, paths []QualifiedPath) ([]Excerpt, error)
	// FindBeforeAndAfter, given a set of possible idsBefore and idsAfter in priority order,
	// finds the immediate previous and next ID with one query
	FindBeforeAndAfter(ctx context.Context, scripture string, idsBefore []string, idsAfter []string) (prev string, next string)
	Search(ctx context.Context, scriptures []string, params SearchParams) ([]Excerpt, error)
}

type BleveExcerptStore struct {
	idx  bleve.Index
	conf *config.DheeConfig
}

// FindBeforeAndAfter implements ExcerptStore.
func (b *BleveExcerptStore) FindBeforeAndAfter(ctx context.Context, scripture string, idsBefore []string, idsAfter []string) (prev string, next string) {
	ids := make([]string, len(idsBefore)+len(idsAfter))
	idset := make(map[string]struct{}, 0)
	for i, p := range idsBefore {
		ids[i] = fmt.Sprintf("%s:%s", scripture, p)
	}
	for i, p := range idsAfter {
		ids[i+len(idsBefore)] = fmt.Sprintf("%s:%s", scripture, p)
	}
	if len(ids) == 0 {
		return "", ""
	}
	query := bleve.NewDocIDQuery(ids)
	searchRequest := bleve.NewSearchRequest(query)
	searchRequest.Size = len(ids)
	searchRequest.Fields = []string{"_id"}

	searchResults, err := b.idx.Search(searchRequest)
	if err != nil {
		slog.Error("error when querying existence of IDs")
		return "", ""
	}
	for _, hit := range searchResults.Hits {
		idset[hit.ID] = struct{}{}
	}
	var before, after string
	for _, id := range idsBefore {
		fullId := fmt.Sprintf("%s:%s", scripture, id)
		if _, found := idset[fullId]; found {
			before = id
			break
		}
	}
	for _, id := range idsAfter {
		fullId := fmt.Sprintf("%s:%s", scripture, id)
		if _, found := idset[fullId]; found {
			after = id
			break
		}
	}
	return before, after
}

// Add implements ExcerptStore.
func (b *BleveExcerptStore) Add(ctx context.Context, scripture string, es []Excerpt) error {
	batch := b.idx.NewBatch()
	for _, e := range es {
		e.Scripture = scripture
		id := fmt.Sprintf("%s:%s", scripture, e.ReadableIndex)
		err := batch.Index(id, e)
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
		ids[i] = fmt.Sprintf("%s:%s", p.Scripture, common.PathToString(p.Path))
	}

	if len(ids) == 0 {
		return nil, nil
	}

	query := bleve.NewDocIDQuery(ids)
	searchRequest := bleve.NewSearchRequest(query)
	searchRequest.Size = len(ids)
	searchRequest.Fields = []string{"*"}

	searchResults, err := b.idx.Search(searchRequest)
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

	// TODO: Support all kinds of search
	var queryMaker func(string, string) query.Query
	switch params.Mode {
	case common.SearchRegex:
		queryMaker = func(q string, field string) query.Query {
			bq := bleve.NewRegexpQuery(q)
			bq.SetField(field)
			return bq
		}
	case common.SearchFuzzy:
		queryMaker = func(q string, field string) query.Query {
			bq := bleve.NewFuzzyQuery(q)
			bq.SetField(field)
			bq.Fuzziness = 2
			return bq
		}
	case common.SearchPrefix:
		queryMaker = func(q string, field string) query.Query {
			bq := bleve.NewPrefixQuery(q)
			bq.SetField(field)
			return bq
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
		sourceQuery := queryMaker(params.Q, "source_text")
		romanQuery := queryMaker(params.Q, "roman_text")
		auxQuery := queryMaker(params.Q, "auxiliaries.*")
		contentQuery = bleve.NewDisjunctionQuery(sourceQuery, romanQuery, auxQuery)
	}

	finalQuery := bleve.NewConjunctionQuery(scriptureQuery, contentQuery)
	searchRequest := bleve.NewSearchRequest(finalQuery)
	searchRequest.Size = 100
	searchRequest.Fields = []string{"*"}
	searchRequest.SortBy([]string{"-_score", "_id"})

	searchResults, err := b.idx.Search(searchRequest)
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

var _ ExcerptStore = &BleveExcerptStore{}

func NewBleveExcerptStore(idx bleve.Index, conf *config.DheeConfig) *BleveExcerptStore {
	return &BleveExcerptStore{idx: idx, conf: conf}
}

// docToExcerpt reconstructs an Excerpt from bleve fields.
// This is complex because of nested structs and slices. Bleve flattens the structure.
// A more robust way would be to store the original JSON and retrieve that.
// For now, we reconstruct it manually, collecting all parsing errors.
func docToExcerpt(fields map[string]any) (Excerpt, error) {
	var e Excerpt
	var errs []error

	var found bool
	var err error

	e.Scripture, found, err = getString(fields, "scripture")
	if err != nil {
		errs = append(errs, err)
	} else if !found {
		errs = append(errs, errors.New("required field 'scripture' not found"))
	}

	e.ReadableIndex, _, err = getString(fields, "readable_index")
	if err != nil {
		errs = append(errs, err)
	}

	e.Meter, _, err = getString(fields, "meter")
	if err != nil {
		errs = append(errs, err)
	}

	e.Path, found, err = getIntSlice(fields, "path")
	if err != nil {
		errs = append(errs, err)
	} else if !found {
		errs = append(errs, errors.New("required field 'path' not found"))
	}

	e.SourceText, found, err = getStringSlice(fields, "source_text")
	if err != nil {
		errs = append(errs, err)
	} else if !found {
		errs = append(errs, errors.New("required field 'source_text' not found"))
	}

	e.RomanText, _, err = getStringSlice(fields, "roman_text")
	if err != nil {
		errs = append(errs, err)
	}

	e.Authors, _, err = getStringSlice(fields, "authors")
	if err != nil {
		errs = append(errs, err)
	}

	e.Addressees, _, err = getStringSlice(fields, "addressees")
	if err != nil {
		errs = append(errs, err)
	}

	e.Group, _, err = getString(fields, "group")
	if err != nil {
		errs = append(errs, err)
	}

	// Reconstruct Auxiliaries
	e.Auxiliaries = make(map[string]Auxiliary)
	for k, v := range fields {
		if strings.HasPrefix(k, "auxiliaries.") && strings.HasSuffix(k, ".text") {
			parts := strings.Split(k, ".")
			if len(parts) == 3 {
				name := parts[1]
				var textSlice []string
				if slice, ok := v.([]any); ok {
					textSlice = make([]string, len(slice))
					for i, t := range slice {
						textSlice[i] = fmt.Sprintf("%v", t)
					}
				} else if v != nil {
					textSlice = []string{fmt.Sprintf("%v", v)}
				}
				e.Auxiliaries[name] = Auxiliary{Text: textSlice}
			}
		}
	}

	// Reconstruct Glossings
	surface, found, err := getStringSlice(fields, "glossings.surface")
	if err != nil {
		errs = append(errs, err)
	} else if found && len(surface) > 0 {
		numGlossings := len(surface)
		glossings := make([]ExcerptGlossing, numGlossings)

		// Helper to get glossing fields and collect errors
		getGlossingField := func(fieldName string) []string {
			slice, _, e := getStringSlice(fields, fieldName)
			if e != nil {
				errs = append(errs, e)
				return nil
			}
			// if f && len(slice) != numGlossings {
			// 	errs = append(errs, fmt.Errorf("field %s has %d elements, expected %d", fieldName, len(slice), numGlossings))
			// 	return nil
			// }
			return slice
		}

		lemma := getGlossingField("glossings.lemma")
		gramm := getGlossingField("glossings.gramm")
		caseField := getGlossingField("glossings.case")
		number := getGlossingField("glossings.number")
		gender := getGlossingField("glossings.gender")
		tense := getGlossingField("glossings.tense")
		voice := getGlossingField("glossings.voice")
		person := getGlossingField("glossings.person")
		mood := getGlossingField("glossings.mood")
		root := getGlossingField("glossings.root")
		modifiers := getGlossingField("glossings.modifiers")

		for i := 0; i < numGlossings; i++ {
			glossings[i].Surface = getSliceElem(surface, i)
			glossings[i].Lemma = getSliceElem(lemma, i)
			glossings[i].Gramm = getSliceElem(gramm, i)
			glossings[i].Case = getSliceElem(caseField, i)
			glossings[i].Number = getSliceElem(number, i)
			glossings[i].Gender = getSliceElem(gender, i)
			glossings[i].Tense = getSliceElem(tense, i)
			glossings[i].Voice = getSliceElem(voice, i)
			glossings[i].Person = getSliceElem(person, i)
			glossings[i].Mood = getSliceElem(mood, i)
			glossings[i].Root = getSliceElem(root, i)
			mod := getSliceElem(modifiers, i)
			if mod != "" {
				glossings[i].Modifiers = []Modifier{Modifier(mod)}
			}
		}
		e.Glossings = [][]ExcerptGlossing{glossings}
	}

	if len(errs) > 0 {
		var sb strings.Builder
		sb.WriteString("failed to deserialize excerpt due to multiple errors:")
		for _, err := range errs {
			sb.WriteString("\n- ")
			sb.WriteString(err.Error())
		}
		return Excerpt{}, errors.New(sb.String())
	}

	return e, nil
}

// getString returns value, found, and error for type mismatch
func getString(fields map[string]any, key string) (string, bool, error) {
	val, ok := fields[key]
	if !ok {
		return "", false, nil
	}
	s, ok := val.(string)
	if !ok {
		return "", true, fmt.Errorf("field '%s' has wrong type: expected string, got %T", key, val)
	}
	return s, true, nil
}

// getStringSlice returns value, found, and error for type mismatch
func getStringSlice(fields map[string]any, key string) ([]string, bool, error) {
	val, ok := fields[key]
	if !ok {
		return nil, false, nil
	}
	if slice, ok := val.([]any); ok {
		res := make([]string, len(slice))
		for i, item := range slice {
			res[i] = fmt.Sprintf("%v", item)
		}
		return res, true, nil
	}
	if s, ok := val.(string); ok {
		return []string{s}, true, nil
	}
	return nil, true, fmt.Errorf("field '%s' has wrong type: expected []string or string, got %T", key, val)
}

// getIntSlice returns value, found, and error for type mismatch
func getIntSlice(fields map[string]any, key string) ([]int, bool, error) {
	val, ok := fields[key]
	if !ok {
		return nil, false, nil
	}
	if slice, ok := val.([]any); ok {
		res := make([]int, len(slice))
		for i, item := range slice {
			f, ok := item.(float64) // JSON numbers are float64
			if !ok {
				return nil, true, fmt.Errorf("expected float64 in slice for key '%s', got %T", key, item)
			}
			res[i] = int(f)
		}
		return res, true, nil
	}
	if f, ok := val.(float64); ok {
		return []int{int(f)}, true, nil
	}
	return nil, true, fmt.Errorf("unsupported type for key '%s': %T", key, val)
}

func getSliceElem(slice []string, i int) string {
	if i < len(slice) {
		return slice[i]
	}
	return ""
}
