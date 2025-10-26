package server

import (
	"html/template"
	"io"

	"github.com/labstack/echo/v4"
	"github.com/mahesh-hegde/dhee/app/config"
)

type TemplateRenderer struct {
	conf *config.DheeConfig
	tmpl *template.Template
}

func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	wrappedData := map[string]any{
		"Page": name,
		"Conf": t.conf,
		"Data": data,
	}
	err := t.tmpl.ExecuteTemplate(w, "layout.html", wrappedData)
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
