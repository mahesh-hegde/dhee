package server

import (
	"html/template"
	"io"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/mahesh-hegde/dhee/app/config"
)

type TemplateRenderer struct {
	conf *config.DheeConfig
	tmpl *template.Template
}

func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	baseName, modifier, found := strings.Cut(name, ".")
	var wrappedData any
	var tName string
	if found && modifier == "preview" {
		wrappedData = data
		// this is a hack which works since all conditions checked in
		// layout.html correspond to filenames
		tName = name + ".html"
	} else {
		wrappedData = map[string]any{
			"Page": baseName,
			"Conf": t.conf,
			"Data": data,
		}
		tName = "layout.html"
	}
	err := t.tmpl.ExecuteTemplate(w, tName, wrappedData)
	if err != nil {
		c.Logger().Error(err)
		return err
	}
	return nil
}

func NewTemplateRenderer(conf *config.DheeConfig) *TemplateRenderer {
	return &TemplateRenderer{
		tmpl: MustParseTemplates(),
		conf: conf,
	}
}
