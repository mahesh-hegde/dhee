package server

import (
	"embed"
	"html/template"
	"strings"
)

//go:embed templ_template/*.ico
var templateFs embed.FS

//go:embed static
var staticFs embed.FS

func sliceOf(a any, others ...any) []any {
	slice := make([]any, 0, len(others)+1)
	slice = append(slice, a)
	slice = append(slice, others...)
	return slice
}

func safe(a any) any {
	if s, ok := a.(string); ok {
		return template.HTML(s)
	}
	if s, ok := a.([]byte); ok {
		return template.HTML(string(s))
	}
	return a
}

func MustParseTemplates() *template.Template {
	funcMap := template.FuncMap{
		"join": strings.Join,
		"sub": func(a, b int) int {
			return a - b
		},
		"sliceOf": sliceOf,
		"safe":    safe,
	}

	return template.Must(template.New("").Funcs(funcMap).ParseFS(templateFs, "template/*.html"))
}
