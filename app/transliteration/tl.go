package transliteration

import (
	"strings"
)

// TlType represents a transliteration scheme type.
type TlType string

const (
	IAST   TlType = "IAST"
	SLP1   TlType = "SLP1"
	HK     TlType = "HK"
	Nagari TlType = "Nagari"
)

// --- Trie structure for efficient prefix matching ---

// use SLP1 as intermediate format for transliteration

// trieNode represents a node in the prefix tree.
type trieNode struct {
	children map[rune]*trieNode
	value    string // The Devanagari value if this node represents the end of a token.
}

// Trie is a prefix tree for storing Roman script to Devanagari mappings.
type Trie struct {
	root *trieNode
}

// newTrie creates a new Trie.
func newTrie() *Trie {
	return &Trie{root: &trieNode{children: make(map[rune]*trieNode)}}
}

// Insert adds a key-value pair to the trie.
func (t *Trie) Insert(key, value string) {
	node := t.root
	for _, r := range key {
		if _, ok := node.children[r]; !ok {
			node.children[r] = &trieNode{children: make(map[rune]*trieNode)}
		}
		node = node.children[r]
	}
	node.value = value
}

// FindLongestPrefix finds the longest registered token that is a prefix of the given string.
// It returns the corresponding Devanagari value and the length of the token found.
func (t *Trie) FindLongestPrefix(s string) (value string, length int) {
	node := t.root
	lastMatchLength := 0
	lastMatchValue := ""

	for i, r := range s {
		if child, ok := node.children[r]; ok {
			node = child
			if node.value != "" {
				lastMatchValue = node.value
				lastMatchLength = i + 1
			}
		} else {
			break // No further matches
		}
	}
	return lastMatchValue, lastMatchLength
}

// --- Transliteration struct and methods ---

// Transliteration handles conversion between different Sanskrit transliteration schemes.
// It pre-computes necessary data structures for efficiency.
type Transliteration struct {
	Fallback []string

	// Private fields for pre-computed data
	romanTries map[TlType]*Trie
	nagariMaps map[TlType]map[string]string
}

// NewTransliteration creates a new Transliteration instance and initializes its data structures.
func NewTransliteration() *Transliteration {
	t := &Transliteration{
		romanTries: make(map[TlType]*Trie),
		nagariMaps: make(map[TlType]map[string]string),
	}

	// Build Tries for Roman to Nagari conversion
	t.romanTries[IAST] = buildTrieFromMap(iastToNagari)
	t.romanTries[SLP1] = buildTrieFromMap(slp1ToNagari)
	t.romanTries[HK] = buildTrieFromMap(hkToNagari)

	// Store maps for Nagari to Roman conversion
	t.nagariMaps[IAST] = nagariToIAST
	t.nagariMaps[SLP1] = nagariToSLP1
	t.nagariMaps[HK] = nagariToHK

	return t
}

func buildTrieFromMap(m map[string]string) *Trie {
	trie := newTrie()
	for k, v := range m {
		trie.Insert(k, v)
	}
	return trie
}

// Convert transliterates a source string from a source scheme to a destination scheme.
func (t *Transliteration) Convert(source string, sourceType TlType, destType TlType) string {
	t.Fallback = nil
	if sourceType == destType {
		return source
	}

	if sourceType != Nagari && destType != Nagari {
		intermediate := t.convertToNagari(source, sourceType)
		return t.convertFromNagari(intermediate, destType)
	}

	if destType == Nagari {
		return t.convertToNagari(source, sourceType)
	}

	return t.convertFromNagari(source, destType)
}

// convertToNagari handles the logic for converting from a Roman script to Devanagari.
func (t *Transliteration) convertToNagari(source string, sourceType TlType) string {
	trie := t.romanTries[sourceType]
	if trie == nil {
		return source
	}

	var result strings.Builder
	var lastRuneAdded rune
	i := 0

	for i < len(source) {
		devanagariChar, length := trie.FindLongestPrefix(source[i:])

		if length > 0 {
			charAsRune := []rune(devanagariChar)[0]

			if devanagariVowels[charAsRune] && lastRuneAdded == virama {
				matra, hasMatra := vowelToMatra[charAsRune]
				// viramaLen := len(string(virama))

				if hasMatra {
					// result.Truncate(result.Len() - viramaLen)
					result.WriteRune(matra)
					lastRuneAdded = matra
				} else { // Handle 'अ' which has no matra by just removing virama
					// result.Truncate(result.Len() - viramaLen)
					// The new last rune is the consonant before the removed virama.
					// This state is complex to track perfectly, so we reset to a neutral value.
					lastRuneAdded = 0
				}
			} else {
				result.WriteString(devanagariChar)
				lastRuneAdded = charAsRune
				if devanagariConsonants[charAsRune] {
					result.WriteRune(virama)
					lastRuneAdded = virama
				}
			}
			i += length
		} else {
			// If no match, append the character as is and add to fallback.
			r := rune(source[i])
			result.WriteRune(r)
			t.Fallback = append(t.Fallback, string(r))
			lastRuneAdded = r
			i++
		}
	}

	return result.String()
}

// convertFromNagari handles logic for converting from Devanagari to a Roman script.
func (t *Transliteration) convertFromNagari(source string, destType TlType) string {
	schemeMap := t.nagariMaps[destType]
	if schemeMap == nil {
		return source
	}

	var result strings.Builder
	runes := []rune(source)

	for i := 0; i < len(runes); i++ {
		char := runes[i]
		charStr := string(char)
		roman, ok := schemeMap[charStr]

		if !ok {
			if vowel, isMatra := matraToVowel[char]; isMatra {
				roman = schemeMap[string(vowel)]
				if result.Len() > 0 && strings.HasSuffix(result.String(), "a") {
					trimmed := result.String()[:result.Len()-1]
					result.Reset()
					result.WriteString(trimmed)
				}
				result.WriteString(roman)
			} else {
				result.WriteString(charStr)
				t.Fallback = append(t.Fallback, charStr)
			}
			continue
		}

		if devanagariConsonants[char] {
			if i+1 < len(runes) && (runes[i+1] == virama || matraToVowel[runes[i+1]] != 0) {
				result.WriteString(roman)
			} else {
				result.WriteString(roman)
				result.WriteString("a")
			}
		} else if char != virama {
			result.WriteString(roman)
		}
	}
	return result.String()
}

// --- Static Data and Initialization ---

var (
	iastToNagari, slp1ToNagari, hkToNagari map[string]string
	devanagariConsonants                   = make(map[rune]bool)
	devanagariVowels                       = make(map[rune]bool)
)

const virama = '्'

var vowelToMatra = map[rune]rune{
	'आ': 'ा', 'इ': 'ि', 'ई': 'ी', 'उ': 'ु', 'ऊ': 'ू', 'ऋ': 'ृ', 'ॠ': 'ॄ',
	'ऌ': 'ॢ', 'ॡ': 'ॣ', 'ए': 'े', 'ऐ': 'ै', 'ओ': 'ो', 'औ': 'ौ',
}

var matraToVowel = map[rune]rune{
	'ा': 'आ', 'ि': 'इ', 'ी': 'ई', 'ु': 'उ', 'ू': 'ऊ', 'ृ': 'ऋ', 'ॄ': 'ॠ',
	'ॢ': 'ऌ', 'ॣ': 'ॡ', 'े': 'ए', 'ै': 'ऐ', 'ो': 'ओ', 'ौ': 'औ',
}

func init() {
	iastToNagari = reverseMap(nagariToIAST)
	slp1ToNagari = reverseMap(nagariToSLP1)
	hkToNagari = reverseMap(nagariToHK)

	for k := range nagariToIASTConsonants {
		devanagariConsonants[[]rune(k)[0]] = true
	}
	for k := range nagariToIASTVowels {
		devanagariVowels[[]rune(k)[0]] = true
	}
}

func reverseMap(m map[string]string) map[string]string {
	reversed := make(map[string]string, len(m))
	for k, v := range m {
		reversed[v] = k
	}
	return reversed
}

var nagariToIASTVowels = map[string]string{
	"अ": "a", "आ": "ā", "इ": "i", "ई": "ī", "उ": "u", "ऊ": "ū", "ऋ": "ṛ", "ॠ": "ṝ",
	"ऌ": "ḷ", "ॡ": "ḹ", "ए": "e", "ऐ": "ai", "ओ": "o", "औ": "au",
}

var (
	nagariToIASTYogavaahas = map[string]string{"ं": "ṃ", "ः": "ḥ", "ँ": "~"}
	nagariToIASTConsonants = map[string]string{
		"क": "k", "ख": "kh", "ग": "g", "घ": "gh", "ङ": "ṅ", "च": "c", "छ": "ch", "ज": "j",
		"झ": "jh", "ञ": "ñ", "ट": "ṭ", "ठ": "ṭh", "ड": "ḍ", "ढ": "ḍh", "ण": "ṇ", "त": "t",
		"थ": "th", "द": "d", "ध": "dh", "न": "n", "प": "p", "फ": "ph", "ब": "b", "भ": "bh",
		"म": "m", "य": "y", "र": "r", "ल": "l", "व": "v", "श": "ś", "ष": "ṣ", "स": "s",
		"ह": "h", "ळ": "ḻ", "क्ष": "kṣ", "ज्ञ": "jñ",
	}
)

var nagariToIASTSymbols = map[string]string{
	"०": "0", "१": "1", "२": "2", "३": "3", "४": "4", "५": "5", "६": "6", "७": "7",
	"८": "8", "९": "9", "ॐ": "oṃ", "ऽ": "'", "।": "|", "॥": "||",
}

var nagariToIAST = func() map[string]string {
	m := make(map[string]string)
	for k, v := range nagariToIASTVowels {
		m[k] = v
	}
	for k, v := range nagariToIASTYogavaahas {
		m[k] = v
	}
	for k, v := range nagariToIASTConsonants {
		m[k] = v
	}
	for k, v := range nagariToIASTSymbols {
		m[k] = v
	}
	return m
}()

var nagariToSLP1 = map[string]string{
	"अ": "a", "आ": "A", "इ": "i", "ई": "I", "उ": "u", "ऊ": "U", "ऋ": "f", "ॠ": "F", "ऌ": "x",
	"ॡ": "X", "ए": "e", "ऐ": "E", "ओ": "o", "औ": "O", "ं": "M", "ः": "H", "ँ": "~", "क": "k",
	"ख": "K", "ग": "g", "घ": "G", "ङ": "N", "च": "c", "छ": "C", "ज": "j", "झ": "J", "ञ": "Y",
	"ट": "w", "ठ": "W", "ड": "q", "ढ": "Q", "ण": "R", "त": "t", "थ": "T", "द": "d", "ध": "D",
	"न": "n", "प": "p", "फ": "P", "ब": "b", "भ": "B", "म": "m", "य": "y", "र": "r", "ल": "l",
	"व": "v", "श": "S", "ष": "z", "स": "s", "ह": "h", "ळ": "L", "क्ष": "kz", "ज्ञ": "jY", "०": "0",
	"१": "1", "२": "2", "३": "3", "४": "4", "५": "5", "६": "6", "७": "7", "८": "8", "९": "9",
	"ॐ": "AUM", "ऽ": "'", "।": ".", "॥": "..",
}

var nagariToHK = map[string]string{
	"अ": "a", "आ": "A", "इ": "i", "ई": "I", "उ": "u", "ऊ": "U", "ऋ": "R", "ॠ": "RR", "ऌ": "lR",
	"ॡ": "lRR", "ए": "e", "ऐ": "ai", "ओ": "o", "औ": "au", "ं": "M", "ः": "H", "ँ": "~", "क": "k",
	"ख": "kh", "ग": "g", "घ": "gh", "ङ": "G", "च": "c", "छ": "ch", "ज": "j", "झ": "jh", "ञ": "J",
	"ट": "T", "ठ": "Th", "ड": "D", "ढ": "Dh", "ण": "N", "त": "t", "थ": "th", "द": "d", "ध": "dh",
	"न": "n", "प": "p", "फ": "ph", "ब": "b", "भ": "bh", "म": "m", "य": "y", "र": "r", "ल": "l",
	"व": "v", "श": "z", "ष": "S", "स": "s", "ह": "h", "ळ": "L", "क्ष": "kS", "ज्ञ": "jJ", "०": "0",
	"१": "1", "२": "2", "३": "3", "४": "4", "५": "5", "६": "6", "७": "7", "८": "8", "९": "9",
	"ॐ": "OM", "ऽ": "'", "।": "|", "॥": "||",
}
