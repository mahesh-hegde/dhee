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

func NormalizeLemma(lemma string) string {
	lemma = strings.TrimSuffix(lemma, "-")
	lemma = strings.TrimRight(lemma, "ⁱ")
	lemma = strings.TrimPrefix(lemma, "√")
	if strings.Contains(lemma, "- ") {
		lemma, _, _ = strings.Cut(lemma, "- ")
	}
	return FoldAccents(lemma)
}

func NormalizeSurface(surface string) string {
	surface = FoldAccents(surface)
	return strings.TrimSuffix(surface, " +")
}

func NormalizePadaWord(pw string) string {
	// MW dictionary has no words containing -
	pw = strings.ReplaceAll(pw, "-", "")
	// PadapATha items often end with " iti"
	pw = strings.TrimSuffix(pw, " iti")
	pw = strings.TrimPrefix(pw, "/ ")
	return FoldAccents(pw)
}
