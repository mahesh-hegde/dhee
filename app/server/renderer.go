//go:generate go tool templ generate
package server

import (
	"io"
	"strings"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
	"github.com/mahesh-hegde/dhee/app/config"
	"github.com/mahesh-hegde/dhee/app/dictionary"
	"github.com/mahesh-hegde/dhee/app/excerpts"
	"github.com/mahesh-hegde/dhee/app/server/templ_template"
)

// PageData wraps template data with page metadata like title.
type PageData struct {
	Title string
	Data  interface{}
}

type TemplateRenderer struct {
	conf   *config.DheeConfig
	hashFS *HashFS
}

type RendererConfigContext struct {
	conf   *config.DheeConfig
	HashFS *HashFS
}

func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	baseName, modifier, found := strings.Cut(name, ".")

	ctx := c.Request().Context()

	// Extract page data and title from PageData if provided
	var pageData interface{}
	pageTitle := t.conf.InstanceName
	if pd, ok := data.(PageData); ok {
		pageData = pd.Data
		if pd.Title != "" {
			pageTitle = pd.Title + " | " + t.conf.InstanceName
		}
	} else {
		pageData = data
	}

	if found && modifier == "preview" {
		var content templ.Component
		switch baseName {
		case "dictionary_search":
			if d, ok := pageData.(dictionary.SearchResults); ok {
				content = templ_template.DictionarySearch(d, true)
			}
		}

		if content == nil {
			content = templ_template.UnsupportedPreview("Page not supported for preview, or invalid data provided.")
		}

		return templ_template.Preview(content).Render(ctx, w)
	}

	var page templ.Component
	switch baseName {
	case "home":
		if d, ok := pageData.(*config.DheeConfig); ok {
			page = templ_template.Home(d)
		}
	case "scripture_search":
		if d, ok := pageData.(*excerpts.ExcerptSearchData); ok {
			page = templ_template.ScriptureSearch(d)
		}
	case "excerpts":
		if d, ok := pageData.(*excerpts.ExcerptTemplateData); ok {
			page = templ_template.Excerpts(d)
		}
	case "dictionary_search":
		if d, ok := pageData.(dictionary.SearchResults); ok {
			page = templ_template.DictionarySearch(d, false)
		}
	case "dictionary_word":
		if d, ok := pageData.(dictionary.DictionaryWordResponse); ok {
			page = templ_template.DictionaryWord(d)
		}
	case "hierarchy":
		if d, ok := pageData.(*excerpts.Hierarchy); ok {
			page = templ_template.Hierarchy(d)
		}
	case "error":
		if d, ok := pageData.(string); ok {
			page = templ_template.Error(d)
		}
	default:
		c.Logger().Errorf("template not found: %s", baseName)
		return echo.ErrNotFound
	}

	if page == nil {
		c.Logger().Errorf("invalid data type for template %s", baseName)
		return echo.ErrInternalServerError
	}

	return templ_template.Layout(config.PageRenderContext{Config: t.conf, AssetHashes: t.hashFS}, pageTitle, page).Render(ctx, w)
}

func NewTemplateRenderer(conf *config.DheeConfig, hashFS *HashFS) *TemplateRenderer {
	return &TemplateRenderer{
		conf:   conf,
		hashFS: hashFS,
	}
}
