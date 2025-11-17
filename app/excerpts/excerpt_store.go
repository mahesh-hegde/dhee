package excerpts

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"

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

func prepareExcerptForDb(conf *config.DheeConfig, e *Excerpt) ExcerptInDB {
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
	translationText := ""
	if sc := conf.GetScriptureByName(e.Scripture); sc != nil && sc.TranslationAuxiliary != "" {
		if aux, ok := e.Auxiliaries[sc.TranslationAuxiliary]; ok {
			translationText = strings.Join(aux.Text, " ")
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
		Translation: translationText,
		Addressees:  e.Addressees,
		Notes:       strings.Join(e.Notes, " "),
		Authors:     e.Authors,
		Meter:       e.Meter,
		Surfaces:    surfaces,
	}
}
