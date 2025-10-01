package dictionary

import (
	"bufio"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// MwXmlEntry represents a single dictionary entry from the XML
type MwXmlEntry struct {
	XMLName xml.Name `xml:""`
	Tag     string   `xml:"-"` // H1, H1A, H2, H3, etc.
	Header  Header   `xml:"h"`
	Body    string   `xml:"body"`
	Tail    Tail     `xml:"tail"`
}

type Header struct {
	Key1 string `xml:"key1"`
	Key2 string `xml:"key2"`
	Hom  string `xml:"hom"`
}

type Tail struct {
	L  string `xml:"L"`
	PC string `xml:"pc"`
}

// DictionaryEntry represents the processed dictionary entry
type DictionaryEntry struct {
	Word            string                `json:"word"`
	OtherSpellings  []string              `json:"other_spellings"`
	PrintedPageNum  string                `json:"printed_page_num"`
	IAST            string                `json:"iast"`
	Body            []DictionaryEntryBody `json:"body"`
	Devanagari      string                `json:"devanagari"`
	HomonymNumber   int                   `json:"homonym_number,omitempty"`
	Stem            string                `json:"stem,omitempty"`
	IsAnimalName    bool                  `json:"is_animal_name,omitempty"`
	IsPlantName     bool                  `json:"is_plant_name,omitempty"`
	LexicalCategory string                `json:"lexical_category,omitempty"`
	Gender          string                `json:"gender,omitempty"`
}

type DictionaryEntryBody struct {
	Plain  string
	Markup string
}

func isEntryTag(tag string) bool {
	// Matches H1, H1A, H2, H2A, H3, etc.
	matched, _ := regexp.MatchString(`^H\d+[A-Z]?$`, tag)
	return matched
}

func xmlToDictionaryEntry(xml MwXmlEntry, lastPageNum string) DictionaryEntry {
	entry := DictionaryEntry{
		Word:           xml.Header.Key1,
		PrintedPageNum: xml.Tail.PC,
	}

	if xml.Header.Key2 != "" && xml.Header.Key2 != entry.Word {
		entry.OtherSpellings = []string{xml.Header.Key2}
	}

	// Use last page number if current entry doesn't have one
	if entry.PrintedPageNum == "" {
		entry.PrintedPageNum = lastPageNum
	}

	// Parse homonym number
	if xml.Header.Hom != "" {
		if num, err := strconv.Atoi(xml.Header.Hom); err == nil {
			entry.HomonymNumber = num
		}
	}

	// Extract other spellings from alternate forms
	otherSpellingRe := regexp.MustCompile(`<info or[^>]*="([^"]+)"`)
	if matches := otherSpellingRe.FindStringSubmatch(xml.Body); matches != nil {
		// Parse format like "L1,X1;L2,X2"
		pairs := strings.Split(matches[1], ";")
		for _, pair := range pairs {
			parts := strings.Split(pair, ",")
			if len(parts) == 2 {
				entry.OtherSpellings = append(entry.OtherSpellings, parts[1])
			}
		}
	}

	// Extract gender information
	genderRe := regexp.MustCompile(`<info lex="([^"]+)"`)
	if matches := genderRe.FindStringSubmatch(xml.Body); matches != nil {
		lex := matches[1]
		if lex != "inh" {
			entry.Gender = lex
		}
	}

	// Extract lexical category
	lexcatRe := regexp.MustCompile(`<info lexcat="([^"]+)"`)
	if matches := lexcatRe.FindStringSubmatch(xml.Body); matches != nil {
		entry.LexicalCategory = matches[1]
	}

	// Extract stem for inflected forms
	stemRe := regexp.MustCompile(`STEM=([^,">]+)`)
	if matches := stemRe.FindStringSubmatch(xml.Body); matches != nil {
		entry.Stem = matches[1]
	}

	// Check for plant/animal names
	// entry.IsPlantName = strings.Contains(xml.Body, "<bot>")
	// entry.IsAnimalName = strings.Contains(xml.Body, "<bio>")

	// Parse body into segments
	entry.Body = parseBody(xml.Body)

	return entry
}

func parseBody(body string) []DictionaryEntryBody {
	var result []DictionaryEntryBody
	tagRe := regexp.MustCompile(`<[^>]+>`)

	lastEnd := 0
	for _, match := range tagRe.FindAllStringIndex(body, -1) {
		start, end := match[0], match[1]

		// Add plain text before tag
		if start > lastEnd {
			plain := body[lastEnd:start]
			if strings.TrimSpace(plain) != "" {
				result = append(result, DictionaryEntryBody{Plain: plain})
			}
		}

		// Add markup
		markup := body[start:end]
		result = append(result, DictionaryEntryBody{Markup: markup})

		lastEnd = end
	}

	// Add remaining plain text
	if lastEnd < len(body) {
		plain := body[lastEnd:]
		if strings.TrimSpace(plain) != "" {
			result = append(result, DictionaryEntryBody{Plain: plain})
		}
	}

	return result
}

func ConvertMonierWilliamsDictionary(inputPath, outputPath string) error {
	// Create output directory
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Open input file
	inFile, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer inFile.Close()

	// Open output file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	writer := bufio.NewWriter(outFile)
	defer writer.Flush()

	// Read and parse XML entries
	decoder := xml.NewDecoder(inFile)
	lastPageNum := ""
	entryCount := 0

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading XML: %w", err)
		}

		if startElem, ok := token.(xml.StartElement); ok {
			// Check if this is an entry element (H1, H1A, H2, etc.)
			if isEntryTag(startElem.Name.Local) {
				var entry MwXmlEntry
				if err := decoder.DecodeElement(&entry, &startElem); err != nil {
					return fmt.Errorf("error decoding entry: %w", err)
				}

				entry.Tag = startElem.Name.Local

				// Convert to dictionary entry
				dictEntry := xmlToDictionaryEntry(entry, lastPageNum)

				// Update last page number
				if dictEntry.PrintedPageNum != "" {
					lastPageNum = dictEntry.PrintedPageNum
				}

				// Write as JSON line
				jsonBytes, err := json.Marshal(dictEntry)
				if err != nil {
					return fmt.Errorf("error marshaling JSON: %w", err)
				}

				if _, err := writer.Write(jsonBytes); err != nil {
					return fmt.Errorf("error writing JSON: %w", err)
				}
				if _, err := writer.WriteString("\n"); err != nil {
					return fmt.Errorf("error writing newline: %w", err)
				}

				entryCount++
				if entryCount%1000 == 0 {
					fmt.Printf("Processed %d entries\n", entryCount)
				}
			}
		}
	}

	fmt.Printf("Conversion complete. Processed %d entries.\n", entryCount)
	return nil
}
