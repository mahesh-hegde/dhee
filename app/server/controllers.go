package server

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/mahesh-hegde/dhee/app/common"
	"github.com/mahesh-hegde/dhee/app/config"
	"github.com/mahesh-hegde/dhee/app/dictionary"
	excerpts "github.com/mahesh-hegde/dhee/app/excerpts"
	"github.com/mahesh-hegde/dhee/app/transliteration"
)

type DheeController struct {
	ds   *dictionary.DictionaryService
	es   *excerpts.ExcerptService
	conf *config.DheeConfig
}

func NewDheeController(dictStore dictionary.DictStore, excerptStore excerpts.ExcerptStore, conf *config.DheeConfig, transliterator *transliteration.Transliterator) *DheeController {
	return &DheeController{
		ds:   dictionary.NewDictionaryService(dictStore, conf, transliterator),
		es:   excerpts.NewExcerptService(dictStore, excerptStore, conf, transliterator),
		conf: conf,
	}
}

func (c *DheeController) GetHome(ctx echo.Context) error {
	return ctx.Render(http.StatusOK, "home", c.conf)
}

func (c *DheeController) GetExcerpts(ctx echo.Context) error {
	scriptureName := ctx.Param("scriptureName")

	pathStr := ctx.Param("path")
	if pathStr == "" {
		pathStr = ctx.QueryParam("path")
		if pathStr != "" {
			return ctx.Redirect(307, ctx.Echo().Reverse("excerpts", scriptureName, pathStr))
		}
	}

	parts := strings.Split(pathStr, ".")

	scri := c.conf.GetScriptureByName(scriptureName)

	if scri == nil {
		return echo.NewHTTPError(404, "invalid text name")
	}

	if len(parts) < len(scri.Hierarchy) {
		return ctx.Redirect(307, ctx.Echo().Reverse("hierarchy", scriptureName, pathStr))
	}

	lastPart := parts[len(parts)-1]

	var paths []excerpts.QualifiedPath
	if strings.Contains(lastPart, "-") {
		rangeParts := strings.Split(lastPart, "-")
		start, err := strconv.Atoi(rangeParts[0])
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid range start")
		}
		end, err := strconv.Atoi(rangeParts[1])
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid range end")
		}

		basePathParts := parts[:len(parts)-1]
		var basePathInts []int
		for _, part := range basePathParts {
			i, err := strconv.Atoi(part)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "Invalid path")
			}
			basePathInts = append(basePathInts, i)
		}

		for i := start; i <= end; i++ {
			pathInts := append(basePathInts, i)
			paths = append(paths, excerpts.QualifiedPath{
				Scripture: scriptureName,
				Path:      pathInts,
			})
		}
	} else {
		var pathInts []int
		for _, part := range parts {
			i, err := strconv.Atoi(part)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "Invalid path")
			}
			pathInts = append(pathInts, i)
		}
		paths = append(paths, excerpts.QualifiedPath{
			Scripture: scriptureName,
			Path:      pathInts,
		})
	}

	excerpts, err := c.es.Get(ctx.Request().Context(), paths)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Failed to get excerpts. Cross check the excerpt number.")
	}

	return ctx.Render(http.StatusOK, "excerpts", excerpts)
}

func (c *DheeController) GetHierarchy(ctx echo.Context) error {
	scriptureName := ctx.Param("scriptureName")
	pathStr := ctx.Param("path")
	var path []int
	if pathStr != "" {
		parts := strings.Split(pathStr, ".")
		for _, part := range parts {
			i, err := strconv.Atoi(part)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "Invalid path")
			}
			path = append(path, i)
		}
	}

	hier, err := c.es.GetHier(ctx.Request().Context(), scriptureName, path)
	if err != nil {
		slog.Error("error getting hierarchy", "err", err)
		return echo.NewHTTPError(http.StatusNotFound, "Failed to get hierarchy")
	}

	return ctx.Render(http.StatusOK, "hierarchy", hier)
}

func (c *DheeController) SearchScripture(ctx echo.Context) error {
	query := ctx.QueryParam("query")
	if query == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "query is required")
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

	params := excerpts.SearchParams{
		Q:          query,
		OriginalQ:  query,
		Tl:         tl,
		Mode:       common.SearchMode(modeStr),
		Scriptures: strings.Split(scriptures, ","),
	}

	excerpts, err := c.es.Search(ctx.Request().Context(), params)
	if err != nil {
		slog.Error("error in scripture search", "err", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to search scripture")
	}

	return ctx.Render(http.StatusOK, "scripture_search", excerpts)
}

func (c *DheeController) GetDictionaryWord(ctx echo.Context) error {
	dictionaryName := ctx.Param("dictionaryName")
	word := ctx.Param("word")

	entries, err := c.ds.GetEntries(ctx.Request().Context(), dictionaryName, []string{word}, common.TlSLP1)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Failed to get dictionary entries")
	}

	return ctx.Render(http.StatusOK, "dictionary_word", entries)
}

func (c *DheeController) SearchDictionary(ctx echo.Context) error {
	dictionaryName := ctx.Param("dictionaryName")
	query := ctx.QueryParam("q")
	textQuery := ctx.QueryParam("textQuery")
	preview := ctx.QueryParam("preview")

	if query == "" && textQuery == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "one of q or textQuery is required")
	}

	if query != "" && textQuery != "" {
		return echo.NewHTTPError(http.StatusBadRequest, "only one of q or textQuery is allowed")
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
		return echo.NewHTTPError(http.StatusBadRequest, "invalid tl value")
	}

	modeStr := ctx.QueryParam("mode")
	if modeStr == "" {
		modeStr = "prefix"
	}

	params := dictionary.SearchParams{
		Query:         query,
		OriginalQuery: query,
		TextQuery:     textQuery,
		Mode:          common.SearchMode(modeStr),
		Tl:            tl,
	}

	results, err := c.ds.Search(ctx.Request().Context(), dictionaryName, params)
	if err != nil {
		return common.WrapErrorForResponse(err, "Failed to search dictionary")
	}

	templateName := "dictionary_search"
	if preview == "true" {
		templateName = "dictionary_search.preview"
	}
	return ctx.Render(http.StatusOK, templateName, results)
}

func (c *DheeController) SuggestDictionary(ctx echo.Context) error {
	dictionaryName := ctx.Param("dictionaryName")
	query := ctx.QueryParam("q")
	if query == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "q is required")
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
		return echo.NewHTTPError(http.StatusBadRequest, "invalid tl value")
	}

	suggestions, err := c.ds.Suggest(ctx.Request().Context(), dictionaryName, query, tl)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get suggestions")
	}

	return ctx.JSON(http.StatusOK, suggestions)
}
