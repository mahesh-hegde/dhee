package server

import (
	"embed"
	"html/template"
)

//go:embed template/*.html
var templateFs embed.FS

func MustParseTemplates() *template.Template {
	return template.Must(template.ParseFS(templateFs, "*.html"))
}
