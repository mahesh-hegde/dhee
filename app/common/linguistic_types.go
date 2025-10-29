package common

import (
	"fmt"
	"strconv"
	"strings"
)

type Language string

const (
	Sanskrit Language = "sanskrit"
	English  Language = "english"
)

// Accent marks will be ignored for search
type Transliteration string

const (
	TlIAST   Transliteration = "iast"
	TlHK     Transliteration = "hk"
	TlNagari Transliteration = "dn"
	TlSLP1   Transliteration = "slp1"
)

type SearchMode string

const (
	SearchExact  SearchMode = "exact"
	SearchRegex  SearchMode = "regex"
	SearchPrefix SearchMode = "prefix"
	SearchFuzzy  SearchMode = "fuzzy"
	SearchASCII  SearchMode = "ascii"
)

func PathToSortString(path []int) string {
	var parts []string
	for _, p := range path {
		parts = append(parts, fmt.Sprintf("%05d", p))
	}
	return strings.Join(parts, ".")
}

func PathToString(path []int) string {
	var parts []string
	for _, p := range path {
		parts = append(parts, strconv.Itoa(p))
	}
	return strings.Join(parts, ".")
}

func StringToPath(pth string) ([]int, error) {
	var parts []int

	for _, p := range strings.Split(pth, ".") {
		idx, err := strconv.Atoi(p)
		if err != nil {
			return nil, err
		}
		parts = append(parts, idx)
	}
	return parts, nil
}
