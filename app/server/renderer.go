//go:generate templ generate
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

type TemplateRenderer struct {
	conf *config.DheeConfig
}

func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	baseName, modifier, found := strings.Cut(name, ".")

	ctx := c.Request().Context()

	if found && modifier == "preview" {
		var content templ.Component
		switch baseName {
		case "dictionary_search":
			if d, ok := data.(dictionary.SearchResults); ok {
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
		if d, ok := data.(*config.DheeConfig); ok {
			page = templ_template.Home(d)
		}
	case "scripture_search":
		if d, ok := data.(*excerpts.ExcerptSearchData); ok {
			page = templ_template.ScriptureSearch(d)
		}
	case "excerpts":
		if d, ok := data.(*excerpts.ExcerptTemplateData); ok {
			page = templ_template.Excerpts(d)
		}
	case "dictionary_search":
		if d, ok := data.(dictionary.SearchResults); ok {
			page = templ_template.DictionarySearch(d, false)
		}
	case "dictionary_word":
		if d, ok := data.(dictionary.DictionaryWordResponse); ok {
			page = templ_template.DictionaryWord(d)
		}
	case "hierarchy":
		if d, ok := data.(*excerpts.Hierarchy); ok {
			page = templ_template.Hierarchy(d)
		}
	case "error":
		if d, ok := data.(string); ok {
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

	pageTitle := t.conf.InstanceName
	if title, ok := c.Get("pageTitle").(string); ok && title != "" {
		pageTitle = title + " - " + t.conf.InstanceName
	}

	return templ_template.Layout(t.conf, pageTitle, page).Render(ctx, w)
}

func NewTemplateRenderer(conf *config.DheeConfig) *TemplateRenderer {
	return &TemplateRenderer{
		conf: conf,
	}
}
