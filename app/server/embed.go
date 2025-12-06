package server

import (
	"crypto/sha256"
	"embed"
	"fmt"
	"html/template"
	"io"
	"strings"
)

//go:embed templ_template/*.ico
var templateFs embed.FS

//go:embed static
var staticFs embed.FS

func hashStaticFile(fileName string) string {
	f, err := staticFs.Open("static/" + fileName)
	if err != nil {
		panic("failed to open static file " + fileName + " for hashing: " + err.Error())
	}
	defer f.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		panic("failed to hash static file " + fileName + ": " + err.Error())
	}
	return fmt.Sprintf("%x", hash.Sum(nil))
}

var commonJsFileHash string = hashStaticFile("common.js")

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
