package server

import (
	"embed"
	"html/template"
	"strings"
)

//go:embed template/*.html
//go:embed template/*.ico
var templateFs embed.FS

func sliceOf(a any, others ...any) []any {
	slice := make([]any, 0, len(others)+1)
	slice = append(slice, a)
	slice = append(slice, others...)
	return slice
}

func MustParseTemplates() *template.Template {
	funcMap := template.FuncMap{
		"join": strings.Join,
		"sub": func(a, b int) int {
			return a - b
		},
		"sliceOf": sliceOf,
	}

	return template.Must(template.New("").Funcs(funcMap).ParseFS(templateFs, "template/*.html"))
}
