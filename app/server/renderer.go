package server

import (
	"html/template"
	"io"

	"github.com/labstack/echo/v4"
)

type TemplateRenderer struct {
	tmpl *template.Template
}

func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	wrappedData := map[string]any{
		"Page": name,
		"Data": data,
	}
	err := t.tmpl.ExecuteTemplate(w, "layout.html", wrappedData)
	if err != nil {
		c.Logger().Error(err)
		return err
	}
	return nil
}

func NewTemplateRenderer() *TemplateRenderer {
	return &TemplateRenderer{
		tmpl: MustParseTemplates(),
	}
}
