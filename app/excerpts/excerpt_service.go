package excerpts

import (
	"context"
	"fmt"
	"html"
	"log/slog"
	"regexp"
	"sort"
	"strings"

	"github.com/mahesh-hegde/dhee/app/common"
	"github.com/mahesh-hegde/dhee/app/config"
	"github.com/mahesh-hegde/dhee/app/dictionary"
	"github.com/mahesh-hegde/dhee/app/transliteration"
)

type ExcerptService struct {
	ds             dictionary.DictStore
	store          ExcerptStore
	conf           *config.DheeConfig
	transliterator *transliteration.Transliterator
	scriptureMap   map[string]config.ScriptureDefn
}

// Get returns the excerpts given by paths. If any of the excerpts could not be found, it returns an error.
//
// Get also batch-fetches the dictionary words for surfaces,
// and lemmas (stripping `-` at end for lemmas). In this process, we expect most entries do not exist
// in the dictionary. We return only those that were found in the batch search.
func (s *ExcerptService) Get(ctx context.Context, paths []QualifiedPath) (*ExcerptTemplateData, error) {
	if len(paths) == 0 || len(paths[0].Path) == 0 {
		return nil, fmt.Errorf("specify a scripture and path")
	}
	excerpts, err := s.store.Get(ctx, paths)
	if err != nil {
		slog.Error("error retrieving excerpts", "err", err)
		return nil, fmt.Errorf("failed to retrieve")
	}

	if len(excerpts) != len(paths) {
		slog.Error("not all excerpts could be found", "err", err)
		return nil, fmt.Errorf("could not find all excerpts")
	}

	// Collect all words to fetch from the dictionary
	wordsToFetch := make(map[string]string)
	padaWordsByExcerpt := make([][]string, len(excerpts))
	for eidx, e := range excerpts {
		for _, g := range e.Glossings {
			for _, gl := range g {
				foldedSurface := common.NormalizeSurface(gl.Surface)
				wordsToFetch[foldedSurface] = ""
				lemma := common.NormalizeLemma(gl.Lemma)
				wordsToFetch[lemma] = ""
			}
		}

		// Do we have Padas?
		padas, hasPadas := e.Auxiliaries["pada"]
		if !hasPadas {
			continue
		}

		padaLines := padas.Text
		padaWords := make([]string, 0, len(padaLines))
		for _, padaLine := range padaLines {
			split := strings.SplitSeq(padaLine, " | ")
			for padaWord := range split {
				padaWord = common.NormalizePadaWord(padaWord)
				padaWords = append(padaWords, strings.TrimSpace(padaWord))
				wordsToFetch[padaWord] = ""
				// if first, _, hyphenated := strings.Cut(padaWord, "-"); hyphenated {
				// 	wordsToFetch[first] = ""
				// }
			}
		}
		padaWordsByExcerpt[eidx] = padaWords
	}

	var words []string

	for w := range wordsToFetch {
		slp1, err := s.transliterator.Convert(w, common.TlIAST, common.TlSLP1)
		if err != nil {
			slog.Debug("could not transliterate word to slp1", "word", w)
		}
		wordsToFetch[w] = slp1
		words = append(words, slp1)
	}

	// Fetch dictionary entries.
	dictName := s.conf.DefaultDict
	dictEntries, err := s.ds.Get(ctx, dictName, words)
	if err != nil {
		return nil, fmt.Errorf("failed to get dictionary entries: %w", err)
	}

	// Map words to their dictionary entries for quick lookup
	wordMap := make(map[string]dictionary.DictionaryEntry)
	for _, entry := range dictEntries {
		wordMap[entry.Word] = entry
	}

	// Combine excerpts with their word meanings
	var es []ExcerptWithWords
	for eidx, e := range excerpts {
		ew := ExcerptWithWords{
			Excerpt: e,
			Words:   make(map[string]dictionary.DictionaryEntry),
			Padas:   make([]PadaElement, 0),
		}
		var glossingPEs []PadaElement
		for _, g := range e.Glossings {
			for _, gl := range g {
				var lemmaMeaning, surfaceMeaning dictionary.DictionaryEntry
				foldedSurface := common.NormalizeSurface(gl.Surface)
				slpWord := wordsToFetch[foldedSurface]
				if entry, ok := wordMap[slpWord]; ok {
					surfaceMeaning = entry
					ew.Words[gl.Surface] = entry
				}
				lemma := common.NormalizeLemma(gl.Lemma)
				slpLemma := wordsToFetch[lemma]
				if entry, ok := wordMap[slpLemma]; ok {
					if slpWord != slpLemma {
						lemmaMeaning = entry
					}
					ew.Words[gl.Lemma] = entry
				}

				glossingPEs = append(glossingPEs, PadaElement{
					Word:            gl.Surface,
					Found:           true,
					G:               gl,
					Slp1NormLemma:   slpLemma,
					Slp1NormSurface: slpWord,
					SurfaceMeaning:  surfaceMeaning,
					LemmaMeaning:    lemmaMeaning,
				})
			}
		}

		padaWords := padaWordsByExcerpt[eidx]
		glossingMap := make(map[string][]int)
		for i, pe := range glossingPEs {
			normalizedSurface := common.NormalizePadaWord(pe.G.Surface)
			glossingMap[normalizedSurface] = append(glossingMap[normalizedSurface], i)
		}

		glossingCursor := 0
		for _, padaWord := range padaWords {
			normPW := common.NormalizePadaWord(padaWord)
			glossingIndex := -1

			// Attempt to match using the map first, looking forward from cursor
			if indices, ok := glossingMap[normPW]; ok {
				for _, idx := range indices {
					if idx >= glossingCursor {
						glossingIndex = idx
						break
					}
				}
			}

			if glossingIndex != -1 {
				// Found a forward match in the map.
				glossingCursor = glossingIndex
				padaElem := glossingPEs[glossingCursor]
				padaElem.Word = padaWord
				padaElem.ExactMatched = true
				ew.Padas = append(ew.Padas, padaElem)
				glossingCursor++
			} else {
				// Fallback to index-based logic
				if glossingCursor >= len(glossingPEs) {
					ew.Padas = append(ew.Padas, PadaElement{
						Word:  padaWord,
						Found: false,
					})
					continue
				}

				padaElem := glossingPEs[glossingCursor]
				padaElem.Word = padaWord
				padaElem.ExactMatched = false // Since it's a fallback
				ew.Padas = append(ew.Padas, padaElem)
				glossingCursor++
			}
		}
		es = append(es, ew)
	}

	sort.Slice(es, func(i, j int) bool {
		p1 := es[i].Path
		p2 := es[j].Path
		for k := 0; k < len(p1) && k < len(p2); k++ {
			if p1[k] != p2[k] {
				return p1[k] < p2[k]
			}
		}
		return len(p1) < len(p2)
	})

	// Calculate possible next and previous candidates
	beforeIds := []string{}
	first := paths[0].Path
	// for now, just consider the last element
	verseIdx := &first[len(first)-1]
	if *verseIdx > 1 {
		*verseIdx -= 1
		beforeIds = append(beforeIds, common.PathToString(first))
		*verseIdx += 1
	}

	up := first[:len(first)-1]

	afterIds := []string{}
	last := paths[len(paths)-1].Path
	if len(last) < 1 {
		return nil, fmt.Errorf("unexpected input in last path element")
	}
	verseIdx = &last[len(last)-1]
	*verseIdx += 1
	afterIds = append(afterIds, common.PathToString(last))
	*verseIdx -= 1

	prev, next := s.store.FindBeforeAndAfter(ctx, paths[0].Scripture, beforeIds, afterIds)

	// Combine excerpt with its scripture information
	scriptureName := paths[0].Scripture
	scri := s.scriptureMap[scriptureName]

	return &ExcerptTemplateData{
		Excerpts:        es,
		AddressedTo:     strings.Join(es[0].Addressees, ", "),
		Scripture:       scri,
		Previous:        prev,
		Next:            next,
		Up:              common.PathToString(up),
		UpType:          scri.Hierarchy[len(up)-1],
		GrammaticalTags: common.GrammaticalTags,
	}, nil
}

// Search returns upto 100 Excerpts which match the search according to search parameters.
func (s *ExcerptService) Search(ctx context.Context, search SearchParams) (*ExcerptSearchData, error) {
	if search.Mode != common.SearchTranslations {
		iastQuery, err := s.transliterator.Convert(search.Q, common.Transliteration(search.Tl), common.TlIAST)
		if err != nil {
			slog.Warn("transliteration failed for scripture search", "query", search.Q, "err", err)
			iastQuery = search.Q
		}
		search.OriginalQ = search.Q
		search.Q = iastQuery
	}

	var re *regexp.Regexp
	var err error
	if search.Mode == "regex" {
		re, err = regexp.Compile("(?s)" + search.Q)
		if err != nil {
			slog.Warn("invalid regex for highlighting", "query", search.Q, "err", err)
		}
	}

	excerpts, err := s.store.Search(ctx, search.Scriptures, search)
	if err != nil {
		return nil, common.WrapErrorForResponse(err, "failed to search excerpts")
	}

	if search.Mode == "regex" {
		for i := range excerpts {
			fullRomanText := strings.Join(excerpts[i].Excerpt.RomanText, "\n")
			escapedText := html.EscapeString(fullRomanText)
			highlightedText := re.ReplaceAllString(escapedText, "<em>$0</em>")
			excerpts[i].RomanHl = highlightedText
		}
	}

	scripture := ""
	if len(search.Scriptures) == 1 {
		scripture = search.Scriptures[0]
	}
	return &ExcerptSearchData{
		Excerpts:  excerpts,
		Search:    search,
		Scripture: *s.conf.GetScriptureByName(scripture),
	}, nil
}

// GetHier returns the hierarchy for a given path.
func (s *ExcerptService) GetHier(ctx context.Context, scriptureName string, path []int) (*Hierarchy, error) {
	scri, ok := s.scriptureMap[scriptureName]
	if !ok {
		return nil, fmt.Errorf("scripture not found: %s", scriptureName)
	}
	return s.store.GetHier(ctx, &scri, path)
}

func NewExcerptService(
	dictStore dictionary.DictStore,
	excerptStore ExcerptStore,
	conf *config.DheeConfig,
	transliterator *transliteration.Transliterator,
) *ExcerptService {
	scriptureMap := map[string]config.ScriptureDefn{}
	for _, scri := range conf.Scriptures {
		scriptureMap[scri.Name] = scri
	}

	return &ExcerptService{
		ds:             dictStore,
		store:          excerptStore,
		conf:           conf,
		transliterator: transliterator,
		scriptureMap:   scriptureMap,
	}
}
