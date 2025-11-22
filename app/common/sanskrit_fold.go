package common

import "strings"

var FoldableAccentsList = []string{
	"ó", "o", "í", "i", "á", "a", "ā́", "ā", "é",
	"e", "ú", "u", "à", "a", "ú", "u",
	"ū́", "ū", "ī́", "ī", "ŕ̥", "ṛ", "r̥", "ṛ", "ṁ", "ṃ", "\u0301", "",
}

var replacer = strings.NewReplacer(FoldableAccentsList...)

func FoldAccents(s string) string {
	return replacer.Replace(s)
}
