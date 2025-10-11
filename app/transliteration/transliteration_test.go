package transliteration

import (
	"testing"

	"github.com/mahesh-hegde/dhee/app/common"
	"github.com/stretchr/testify/assert"
)

func TestTransliterator_Convert(t *testing.T) {
	options := TlOptions{}
	transliterator, err := NewTransliterator(options)
	assert.NoError(t, err)

	testCases := []struct {
		name      string
		source    string
		sourceTl  common.Transliteration
		targetTl  common.Transliteration
		expected  string
		expectErr bool
	}{
		// SLP1 to others
		{"SLP1 to IAST", "saMskfta", common.TlSLP1, common.TlIAST, "saṃskṛta", false},
		{"SLP1 to HK", "saMskfta", common.TlSLP1, common.TlHK, "saMskRta", false},
		{"SLP1 to Devanagari", "saMskftam", common.TlSLP1, common.TlNagari, "संस्कृतम्", false},
		{"SLP1 to Devanagari Complex", "rAmaH kfzRaSca", common.TlSLP1, common.TlNagari, "रामः कृष्णश्च", false},
		{"SLP1 to Devanagari Vowel Start", "indra", common.TlSLP1, common.TlNagari, "इन्द्र", false},

		// Others to SLP1
		{"IAST to SLP1", "saṃskṛta", common.TlIAST, common.TlSLP1, "saMskfta", false},
		{"HK to SLP1", "saMskRta", common.TlHK, common.TlSLP1, "saMskfta", false},
		{"Devanagari to SLP1", "संस्कृतम्", common.TlNagari, common.TlSLP1, "saMskftam", false},
		{"Devanagari to SLP1 Complex", "रामः कृष्णश्च", common.TlNagari, common.TlSLP1, "rAmaH kfzRaSca", false},
		{"Devanagari to SLP1 Vowel Start", "इन्द्र", common.TlNagari, common.TlSLP1, "indra", false},

		// Non-SLP1 to Non-SLP1
		{"IAST to Devanagari", "saṃskṛtam", common.TlIAST, common.TlNagari, "संस्कृतम्", false},
		{"Devanagari to HK", "संस्कृतम्", common.TlNagari, common.TlHK, "saMskRtam", false},
		{"HK to IAST", "saMskRta", common.TlHK, common.TlIAST, "saṃskṛta", false},

		// Identity conversion
		{"IAST to IAST", "saṃskṛta", common.TlIAST, common.TlIAST, "saṃskṛta", false},
		{"SLP1 to SLP1", "saMskfta", common.TlSLP1, common.TlSLP1, "saMskfta", false},

		// Edge cases
		{"Empty String", "", common.TlSLP1, common.TlIAST, "", false},
		{"Unsupported Source", "test", "unsupported", common.TlIAST, "", true},
		{"Unsupported Target", "test", common.TlIAST, "unsupported", "", true},
		{"Unmapped characters", "abc_123", common.TlSLP1, common.TlIAST, "abc_123", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := transliterator.Convert(tc.source, tc.sourceTl, tc.targetTl)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, actual)
			}
		})
	}
}

func TestTransliterator_Convert_WithFallback(t *testing.T) {
	options := TlOptions{FallbackCharacter: "?"}
	transliterator, err := NewTransliterator(options)
	assert.NoError(t, err)

	testCases := []struct {
		name     string
		source   string
		sourceTl common.Transliteration
		targetTl common.Transliteration
		expected string
	}{
		{"Unmapped with fallback", "abc_123", common.TlSLP1, common.TlIAST, "abc?123"},
		{"Unmapped Devanagari with fallback", "अ_ब", common.TlNagari, common.TlSLP1, "a?ba"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := transliterator.Convert(tc.source, tc.sourceTl, tc.targetTl)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
