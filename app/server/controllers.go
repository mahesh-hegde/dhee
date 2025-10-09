package server

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/blevesearch/bleve/v2"
	"github.com/labstack/echo/v4"
	"github.com/mahesh-hegde/dhee/app/common"
	"github.com/mahesh-hegde/dhee/app/config"
	"github.com/mahesh-hegde/dhee/app/dictionary"
	"github.com/mahesh-hegde/dhee/app/scripture"
)

type DheeController struct {
	ds *dictionary.DictionaryService
	es *scripture.ExcerptService
}

func NewDheeController(index bleve.Index, conf *config.DheeConfig) *DheeController {
	return &DheeController{
		ds: dictionary.NewDictionaryService(index, conf),
		es: scripture.NewScriptureService(index, conf),
	}
}

func (c *DheeController) GetExcerpts(ctx echo.Context) error {
	scriptureName := ctx.Param("scriptureName")
	pathStr := ctx.Param("path")

	parts := strings.Split(pathStr, ".")
	lastPart := parts[len(parts)-1]

	var paths []scripture.QualifiedPath
	if strings.Contains(lastPart, "-") {
		rangeParts := strings.Split(lastPart, "-")
		start, err := strconv.Atoi(rangeParts[0])
		if err != nil {
			return ctx.String(http.StatusBadRequest, "Invalid range start")
		}
		end, err := strconv.Atoi(rangeParts[1])
		if err != nil {
			return ctx.String(http.StatusBadRequest, "Invalid range end")
		}

		basePathParts := parts[:len(parts)-1]
		var basePathInts []int
		for _, part := range basePathParts {
			i, err := strconv.Atoi(part)
			if err != nil {
				return ctx.String(http.StatusBadRequest, "Invalid path")
			}
			basePathInts = append(basePathInts, i)
		}

		for i := start; i <= end; i++ {
			pathInts := append(basePathInts, i)
			paths = append(paths, scripture.QualifiedPath{
				Scripture: scriptureName,
				Path:      pathInts,
			})
		}
	} else {
		var pathInts []int
		for _, part := range parts {
			i, err := strconv.Atoi(part)
			if err != nil {
				return ctx.String(http.StatusBadRequest, "Invalid path")
			}
			pathInts = append(pathInts, i)
		}
		paths = append(paths, scripture.QualifiedPath{
			Scripture: scriptureName,
			Path:      pathInts,
		})
	}

	excerpts, err := c.es.Get(ctx.Request().Context(), paths)
	if err != nil {
		return ctx.String(http.StatusInternalServerError, "Failed to get excerpts")
	}

	return ctx.Render(http.StatusOK, "excerpts.html", excerpts)
}

func (c *DheeController) SearchScripture(ctx echo.Context) error {
	query := ctx.QueryParam("query")
	if query == "" {
		return ctx.String(http.StatusBadRequest, "query is required")
	}

	tl := ctx.QueryParam("tl")
	if tl == "" {
		tl = "slp1"
	}

	modeStr := ctx.QueryParam("mode")
	if modeStr == "" {
		modeStr = "exact"
	}

	scriptures := ctx.QueryParam("scriptures")

	params := scripture.SearchParams{
		Q:          query,
		Tl:         tl,
		Mode:       common.SearchMode(modeStr),
		Scriptures: strings.Split(scriptures, ","),
	}

	excerpts, err := c.es.Search(ctx.Request().Context(), params)
	if err != nil {
		return ctx.String(http.StatusInternalServerError, "Failed to search scripture")
	}

	return ctx.Render(http.StatusOK, "scripture_search.html", excerpts)
}

func (c *DheeController) GetDictionaryWord(ctx echo.Context) error {
	dictionaryName := ctx.Param("dictionaryName")
	word := ctx.Param("word")

	entries, err := c.ds.GetEntries(ctx.Request().Context(), dictionaryName, []string{word}, common.TlSLP1)
	if err != nil {
		return ctx.String(http.StatusInternalServerError, "Failed to get dictionary entries")
	}

	return ctx.Render(http.StatusOK, "dictionary_word.html", entries)
}

func (c *DheeController) SearchDictionary(ctx echo.Context) error {
	dictionaryName := ctx.Param("dictionaryName")
	query := ctx.QueryParam("q")
	textQuery := ctx.QueryParam("textQuery")

	if query == "" && textQuery == "" {
		return ctx.String(http.StatusBadRequest, "one of q or textQuery is required")
	}

	if query != "" && textQuery != "" {
		return ctx.String(http.StatusBadRequest, "only one of q or textQuery is allowed")
	}

	tlStr := ctx.QueryParam("tl")
	if tlStr == "" {
		tlStr = "slp1"
	}

	var tl common.Transliteration
	switch tlStr {
	case "iast":
		tl = common.TlIAST
	case "hk":
		tl = common.TlHK
	case "dn":
		tl = common.TlNagari
	case "slp1":
		tl = common.TlSLP1
	default:
		return ctx.String(http.StatusBadRequest, "invalid tl value")
	}

	modeStr := ctx.QueryParam("mode")
	if modeStr == "" {
		modeStr = "prefix"
	}

	params := dictionary.SearchParams{
		Query:     query,
		TextQuery: textQuery,
		Mode:      common.SearchMode(modeStr),
		Tl:        tl,
	}

	results, err := c.ds.Search(ctx.Request().Context(), dictionaryName, params)
	if err != nil {
		return ctx.String(http.StatusInternalServerError, "Failed to search dictionary")
	}

	return ctx.Render(http.StatusOK, "dictionary_search.html", results)
}

func (c *DheeController) SuggestDictionary(ctx echo.Context) error {
	dictionaryName := ctx.Param("dictionaryName")
	query := ctx.QueryParam("q")
	if query == "" {
		return ctx.String(http.StatusBadRequest, "q is required")
	}

	tlStr := ctx.QueryParam("tl")
	if tlStr == "" {
		tlStr = "slp1"
	}

	var tl common.Transliteration
	switch tlStr {
	case "iast":
		tl = common.TlIAST
	case "hk":
		tl = common.TlHK
	case "dn":
		tl = common.TlNagari
	case "slp1":
		tl = common.TlSLP1
	default:
		return ctx.String(http.StatusBadRequest, "invalid tl value")
	}

	suggestions, err := c.ds.Suggest(ctx.Request().Context(), dictionaryName, query, tl)
	if err != nil {
		return ctx.String(http.StatusInternalServerError, "Failed to get suggestions")
	}

	return ctx.JSON(http.StatusOK, suggestions)
}
