package server

import (
	"embed"
	"html/template"
	"strings"
)

//go:embed template/*.html
var templateFs embed.FS

func MustParseTemplates() *template.Template {
	funcMap := template.FuncMap{
		"join": strings.Join,
		"sub": func(a, b int) int {
			return a - b
		},
	}

	return template.Must(template.New("").Funcs(funcMap).ParseFS(templateFs, "template/*.html"))
}
