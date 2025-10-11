package transliteration

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/mahesh-hegde/dhee/app/common"
)

//go:embed slp1_mappings.json
var slp1MappingsFile []byte

// TlOptions provides user-configurable options for transliteration.
type TlOptions struct {
	// FallbackCharacter is used when a character cannot be transliterated.
	// If empty, the original character is retained.
	FallbackCharacter string
}

// slp1MappingGroup defines the structure for a group of mappings (e.g., vowels, consonants).
type slp1MappingGroup map[string]string

// slp1SchemeMap defines the structure for a single transliteration scheme (e.g., slp1_to_iast).
type slp1SchemeMap struct {
	Vowels          slp1MappingGroup `json:"vowels"`
	Yogavaahas      slp1MappingGroup `json:"yogavaahas"`
	Virama          slp1MappingGroup `json:"virama"`
	Consonants      slp1MappingGroup `json:"consonants"`
	Symbols         slp1MappingGroup `json:"symbols"`
	Accents         slp1MappingGroup `json:"accents"`
	ExtraConsonants slp1MappingGroup `json:"extra_consonants"`
	Shortcuts       slp1MappingGroup `json:"shortcuts"`
}

// slp1MappingsJSON is the top-level structure for the embedded JSON file.
type slp1MappingsJSON struct {
	Mappings map[string]slp1SchemeMap `json:"mappings"`
}

// KeyMap is a map of keys grouped by their first character for efficient lookup.
// The keys in the inner slice are sorted by length in descending order.
type KeyMap map[string][]string

// SchemeKeyMap maps a transliteration scheme to its KeyMap.
type SchemeKeyMap map[common.Transliteration]KeyMap

// Transliterator provides sanskrit transliteration functionality.
type Transliterator struct {
	options        TlOptions
	fromSlp1       map[common.Transliteration]map[string]string
	toSlp1         map[common.Transliteration]map[string]string
	keys           SchemeKeyMap // For longest-match search
	slp1Vowels     map[string]bool
	slp1Consonants map[string]bool
}

// Convert converts a string between defined transliteration formats.
func (t *Transliterator) Convert(source string, sourceTl common.Transliteration, targetTl common.Transliteration) (string, error) {
	if sourceTl == targetTl {
		return source, nil
	}

	// SLP1 is the intermediate format.
	var slp1Text string
	var err error

	if sourceTl == common.TlSLP1 {
		slp1Text = source
	} else {
		sourceMap, ok := t.toSlp1[sourceTl]
		if !ok {
			return "", fmt.Errorf("unsupported source transliteration: %s", sourceTl)
		}
		if sourceTl == common.TlNagari {
			slp1Text = t.doConvertFromDevanagari(source, sourceMap)
		} else {
			slp1Text = t.doConvert(source, sourceMap, t.keys[sourceTl])
		}
	}

	if targetTl == common.TlSLP1 {
		return slp1Text, nil
	}

	targetMap, ok := t.fromSlp1[targetTl]
	if !ok {
		return "", fmt.Errorf("unsupported target transliteration: %s", targetTl)
	}

	var result string
	if targetTl == common.TlNagari {
		result = t.doConvertDevanagari(slp1Text, targetMap, t.keys[common.TlSLP1])
	} else {
		result = t.doConvert(slp1Text, targetMap, t.keys[common.TlSLP1])
	}

	return result, err
}

// findLongestMatch finds the longest key from the KeyMap that is a prefix of the source string at the given offset.
func findLongestMatch(source string, offset int, keyMap KeyMap) string {
	if offset >= len(source) {
		return ""
	}

	// Use runes to correctly handle multi-byte characters which might be the first character.
	firstChar := string([]rune(source[offset:])[0])

	// Get the candidate keys for this starting character.
	sortedKeys, ok := keyMap[firstChar]
	if !ok {
		return ""
	}

	// The keys are pre-sorted by length, descending.
	for _, key := range sortedKeys {
		if strings.HasPrefix(source[offset:], key) {
			return key
		}
	}
	return ""
}

// doConvertDevanagari handles the specific rules for Devanagari script generation.
func (t *Transliterator) doConvertDevanagari(source string, convMap map[string]string, keyMap KeyMap) string {
	var result strings.Builder
	i := 0
	sourceLen := len(source)

	vowelToMatra := map[string]string{
		"A": "ा", "i": "ि", "I": "ी", "u": "ु", "U": "ू",
		"f": "ृ", "F": "ॄ", "x": "ॢ", "X": "ॣ", "e": "े",
		"E": "ै", "o": "ो", "O": "ौ",
	}

	for i < sourceLen {
		match := findLongestMatch(source, i, keyMap)

		if match == "" {
			r, size := utf8.DecodeRuneInString(source[i:])
			if t.options.FallbackCharacter != "" {
				result.WriteString(t.options.FallbackCharacter)
			} else {
				result.WriteRune(r)
			}
			i += size
			continue
		}

		isConsonant := t.slp1Consonants[match]
		isVowel := t.slp1Vowels[match]

		if isConsonant {
			result.WriteString(convMap[match])

			// Look ahead for a vowel to form a matra
			nextMatch := findLongestMatch(source, i+len(match), keyMap)
			isNextVowel := t.slp1Vowels[nextMatch]

			if isNextVowel {
				if matra, ok := vowelToMatra[nextMatch]; ok {
					result.WriteString(matra)
				}
				// For 'a', no matra is needed.
				i += len(match) + len(nextMatch) // Consume both consonant and vowel
			} else {
				// No vowel follows, so add a virama
				result.WriteString(convMap[""]) // convMap[""] should be "्"
				i += len(match)
			}
		} else if isVowel {
			// This is a standalone vowel (e.g., at the beginning of a word)
			result.WriteString(convMap[match])
			i += len(match)
		} else {
			// It's a symbol, yogavaaha, etc.
			result.WriteString(convMap[match])
			i += len(match)
		}
	}
	return result.String()
}

// doConvertFromDevanagari handles the specific rules for converting Devanagari to SLP1.
func (t *Transliterator) doConvertFromDevanagari(source string, convMap map[string]string) string {
	var result strings.Builder
	sourceRunes := []rune(source)

	// We need a reverse map for matras
	matraToVowel := map[rune]string{
		'ा': "A", 'ि': "i", 'ी': "I", 'ु': "u", 'ू': "U",
		'ृ': "f", 'ॄ': "F", 'ॢ': "x", 'ॣ': "X", 'े': "e",
		'ै': "E", 'ो': "o", 'ौ': "O",
	}
	virama := '्'

	i := 0
	for i < len(sourceRunes) {
		char := sourceRunes[i]
		slp1Char, isMapped := convMap[string(char)]

		if isMapped {
			// It's a consonant, full vowel, or symbol.
			isConsonant := t.slp1Consonants[slp1Char]

			if isConsonant {
				// Look ahead for matra or virama
				if i+1 < len(sourceRunes) {
					nextChar := sourceRunes[i+1]
					if vowel, isMatra := matraToVowel[nextChar]; isMatra {
						result.WriteString(slp1Char)
						result.WriteString(vowel)
						i += 2 // Consume consonant and matra
						continue
					} else if nextChar == virama {
						result.WriteString(slp1Char)
						i += 2 // Consume consonant and virama
						continue
					}
				}
				// No matra or virama, so it has an inherent 'a'
				result.WriteString(slp1Char)
				result.WriteString("a")
				i++
			} else {
				// It's a full vowel or a symbol, just write it
				result.WriteString(slp1Char)
				i++
			}
		} else {
			// Unmapped character
			if t.options.FallbackCharacter != "" {
				result.WriteString(t.options.FallbackCharacter)
			} else {
				result.WriteRune(char)
			}
			i++
		}
	}
	return result.String()
}

// doConvert performs the core transliteration using a pre-computed map.
// It finds the longest matching key at each position in the source string.
func (t *Transliterator) doConvert(source string, convMap map[string]string, keyMap KeyMap) string {
	var result strings.Builder
	i := 0
	sourceLen := len(source)
	for i < sourceLen {
		match := findLongestMatch(source, i, keyMap)

		if match != "" {
			result.WriteString(convMap[match])
			i += len(match)
		} else {
			r, size := utf8.DecodeRuneInString(source[i:])
			if t.options.FallbackCharacter != "" {
				result.WriteString(t.options.FallbackCharacter)
			} else {
				result.WriteRune(r)
			}
			i += size
		}
	}
	return result.String()
}

// NewTransliterator returns a fully initialized transliterator with given options.
func NewTransliterator(options TlOptions) (*Transliterator, error) {
	var mappingsData slp1MappingsJSON
	if err := json.Unmarshal(slp1MappingsFile, &mappingsData); err != nil {
		return nil, fmt.Errorf("failed to parse embedded slp1_mappings.json: %w", err)
	}

	t := &Transliterator{
		options:        options,
		fromSlp1:       make(map[common.Transliteration]map[string]string),
		toSlp1:         make(map[common.Transliteration]map[string]string),
		keys:           make(SchemeKeyMap),
		slp1Vowels:     make(map[string]bool),
		slp1Consonants: make(map[string]bool),
	}

	// Add SLP1 to itself mapping
	t.fromSlp1[common.TlSLP1] = map[string]string{}
	t.toSlp1[common.TlSLP1] = map[string]string{}

	// Populate character type maps (vowels, consonants) from the "slp1_to_iast" mapping,
	// which is a reliable source for character classification.
	if baseMap, ok := mappingsData.Mappings["slp1_to_iast"]; ok {
		for k := range baseMap.Vowels {
			t.slp1Vowels[k] = true
		}
		for k := range baseMap.Consonants {
			t.slp1Consonants[k] = true
		}
		for k := range baseMap.ExtraConsonants {
			t.slp1Consonants[k] = true // Treat extra consonants as such
		}
	} else {
		return nil, fmt.Errorf("base 'slp1_to_iast' mapping not found for character classification")
	}

	for schemeName, schemeMap := range mappingsData.Mappings {
		parts := strings.Split(schemeName, "_to_")
		if len(parts) != 2 {
			continue
		}
		targetSchemeStr := parts[1]
		if targetSchemeStr == "devanagari" {
			targetSchemeStr = "dn"
		}
		targetScheme := common.Transliteration(targetSchemeStr)

		fromMap := make(map[string]string)
		toMap := make(map[string]string)

		groups := []slp1MappingGroup{
			schemeMap.Vowels, schemeMap.Yogavaahas, schemeMap.Virama,
			schemeMap.Consonants, schemeMap.Symbols, schemeMap.Accents,
			schemeMap.ExtraConsonants, schemeMap.Shortcuts,
		}

		for _, group := range groups {
			for slp1Char, targetChar := range group {
				fromMap[slp1Char] = targetChar
				if slp1Char != "" {
					toMap[targetChar] = slp1Char
				}
			}
		}
		t.fromSlp1[targetScheme] = fromMap
		t.toSlp1[targetScheme] = toMap
	}

	// Pre-compute key maps for longest-match search
	for scheme, convMap := range t.toSlp1 {
		keyMap := make(KeyMap)
		for k := range convMap {
			if k == "" {
				continue
			}
			firstChar := string([]rune(k)[0])
			keyMap[firstChar] = append(keyMap[firstChar], k)
		}
		for _, keys := range keyMap {
			sort.Slice(keys, func(i, j int) bool {
				return len(keys[i]) > len(keys[j])
			})
		}
		t.keys[scheme] = keyMap
	}

	// Pre-compute keys for from-slp1 maps (keys are slp1)
	slp1KeyMap := make(KeyMap)
	for _, convMap := range t.fromSlp1 {
		for k := range convMap {
			if k == "" {
				continue
			}
			firstChar := string(k[0])
			// Avoid duplicates
			found := false
			for _, existingKey := range slp1KeyMap[firstChar] {
				if existingKey == k {
					found = true
					break
				}
			}
			if !found {
				slp1KeyMap[firstChar] = append(slp1KeyMap[firstChar], k)
			}
		}
	}
	for _, keys := range slp1KeyMap {
		sort.Slice(keys, func(i, j int) bool {
			return len(keys[i]) > len(keys[j])
		})
	}
	t.keys[common.TlSLP1] = slp1KeyMap

	return t, nil
}
