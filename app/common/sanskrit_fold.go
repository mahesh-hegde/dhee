package common

import "strings"

var FoldableAccents = map[string]string{
	"ó":  "o",
	"í":  "i",
	"á":  "a",
	"ā́": "ā",
	"é":  "e",
	"ú":  "u",
	"à":  "a",
}

var FoldableAccentsList = []string{"ó", "o", "í", "i", "á", "a", "ā́", "ā", "é", "e", "ú", "u", "à", "a"}

var replacer = strings.NewReplacer(FoldableAccentsList...)

func FoldAccents(s string) string {
	return replacer.Replace(s)
}
