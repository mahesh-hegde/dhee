package dictionary

import (
	"bufio"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

type MwXmlEntryBody struct {
	Content string `xml:",innerxml"`
}

// MwXmlEntry represents a single dictionary entry from the XML
type MwXmlEntry struct {
	XMLName xml.Name       `xml:""`
	Tag     string         `xml:"-"` // H1, H1A, H2, H3, etc.
	Header  Header         `xml:"h"`
	Body    MwXmlEntryBody `xml:"body"`
	Tail    Tail           `xml:"tail"`
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

func isEntryTag(tag string) bool {
	// Matches H1, H1A, H2, H2A, H3, etc.
	matched, _ := regexp.MatchString(`^H\d+[A-Z]?$`, tag)
	return matched
}

func xmlToDictionaryEntry(xml MwXmlEntry, lastPageNum string) DictionaryEntry {
	entry := DictionaryEntry{
		Word:           xml.Header.Key1,
		PrintedPageNum: xml.Tail.PC,
		HTag:           xml.Tag,
		Id:             xml.Tail.L,
	}

	if xml.Header.Key2 != "" && xml.Header.Key2 != entry.Word {
		entry.Variants = []string{xml.Header.Key2}
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

	// Parse body into segments
	parseBody(xml.Body.Content, &entry)

	return entry
}

func parseBody(body string, entry *DictionaryEntry) {
	entry.Body.Markup = body

	// Wrap in root element to create valid XML
	wrapped := "<root>" + body + "</root>"

	decoder := xml.NewDecoder(strings.NewReader(wrapped))
	decoder.Strict = false // tolerate malformed XML
	decoder.Token()

	var plainText strings.Builder

	if err := walkXMLTree(decoder, &plainText, entry, 0); err != nil {
		slog.Warn("error parsing body", "error", err, "id", entry.Id)
	}

	entry.Body.Plain = strings.TrimSpace(plainText.String())
}

func walkXMLTree(decoder *xml.Decoder, plainText *strings.Builder, entry *DictionaryEntry, depth int) error {
	if depth > 5 {
		return nil
	}

	for {
		token, err := decoder.Token()
		if err != nil {
			if !errors.Is(err, io.EOF) {
				slog.Info("error when walking XML", "err", err)
			}
			return nil // EOF or error, stop gracefully
		}

		switch elem := token.(type) {
		case xml.StartElement:
			if err := handleStartElement(elem, decoder, plainText, entry, depth); err != nil {
				if !errors.Is(err, io.EOF) {
					slog.Warn("error handling element", "tag", elem.Name.Local, "error", err)
				}
			}

		case xml.CharData:
			plainText.Write(elem)

		case xml.EndElement:
			return nil
		}
	}
}

func handleStartElement(elem xml.StartElement, decoder *xml.Decoder, plainText *strings.Builder, entry *DictionaryEntry, depth int) error {
	tagName := elem.Name.Local
	switch tagName {
	case "ab":
		return handleAb(elem, decoder, plainText)

	case "s1":
		return handleS1(elem, decoder, plainText)

	case "s":
		return handleS(elem, decoder, plainText, entry)

	case "ls":
		return handleLs(decoder, plainText, entry)

	case "hom":
		return handleHom(decoder, plainText, entry)

	case "etym", "lang", "bot", "bio", "ns", "i", "lex":
		return handleStripTag(decoder, plainText)

	case "info":
		return handleInfo(elem, entry)

	case "pb", "div", "pcol":
		return handlePcol(decoder, plainText)

	case "shortlong", "srs":
		// These appear within <s> tags, skip them
		return nil

	default:
		// Recurse into unknown tags
		return walkXMLTree(decoder, plainText, entry, depth+1)
	}
}

func handleAb(elem xml.StartElement, decoder *xml.Decoder, plainText *strings.Builder) error {
	var slp1, expansion string
	for _, attr := range elem.Attr {
		switch attr.Name.Local {
		case "slp1":
			slp1 = attr.Value
		case "n":
			expansion = attr.Value
		}
	}

	// Read content
	content, err := readElementText(decoder)
	if err != nil {
		return err
	}

	// Use expansion if available, otherwise strip
	if expansion != "" {
		plainText.WriteString(expansion)
	} else if slp1 != "" {
		plainText.WriteString(slp1)
	} else {
		plainText.WriteString(content)
	}

	return nil
}

func handleS1(_ xml.StartElement, decoder *xml.Decoder, plainText *strings.Builder) error {
	content, err := readElementText(decoder)
	if err != nil {
		return err
	}
	plainText.WriteString(content)
	return nil
}

func handleS(_ xml.StartElement, decoder *xml.Decoder, plainText *strings.Builder, entry *DictionaryEntry) error {
	content, err := readElementText(decoder)
	if err != nil {
		return err
	}
	// Remove if equal to word or in otherSpellings
	if content != entry.Word {
		found := slices.Contains(entry.Variants, content)
		// TODO: convert to IAST always
		if !found {
			plainText.WriteString(content)
		}
	}
	return nil
}

func handleLs(decoder *xml.Decoder, plainText *strings.Builder, entry *DictionaryEntry) error {
	content, err := readElementText(decoder)
	if err != nil {
		return err
	}
	plainText.WriteRune('[')
	plainText.WriteString(content)
	plainText.WriteRune(']')
	entry.LitRefs = append(entry.LitRefs, content)
	return nil
}

func handleHom(decoder *xml.Decoder, plainText *strings.Builder, entry *DictionaryEntry) error {
	content, err := readElementText(decoder)
	if err != nil {
		return err
	}

	// Remove if at beginning and matches HomonymNumber
	trimmed := strings.TrimSuffix(strings.TrimSpace(content), ".")
	if entry.HomonymNumber > 0 && trimmed == strconv.Itoa(entry.HomonymNumber) {
		// Skip it
		return nil
	}

	plainText.WriteString(content)
	return nil
}

func handleStripTag(decoder *xml.Decoder, plainText *strings.Builder) error {
	content, err := readElementText(decoder)
	if err != nil {
		return err
	}

	// For etym, lang, bot, bio - add to cognates or just strip
	plainText.WriteString(content)
	return nil
}

func handlePcol(decoder *xml.Decoder, plainText *strings.Builder) error {
	content, err := readElementText(decoder)
	if err != nil {
		return err
	}
	plainText.WriteString(content)
	return nil
}

func handleInfo(elem xml.StartElement, entry *DictionaryEntry) error {
	for _, attr := range elem.Attr {
		switch attr.Name.Local {
		case "lex":
			entry.LexicalGender = attr.Value

		case "lexcat":
			parseLexCat(attr.Value, entry)

		case "verb":
			parseVerb(elem.Attr, entry)

		case "or", "and", "orsl", "orwr":
			parseOtherSpellings(attr.Value, entry)
		}
	}

	return skipElement(xml.NewDecoder(strings.NewReader("<dummy/>")))
}

func parseLexCat(value string, entry *DictionaryEntry) {
	if value == "loan" {
		entry.LexCat = LexCat{IsLoan: true}
		return
	}

	// Parse LEXID=X,STEM=Y format
	parts := strings.Split(value, ",")
	var lexCat LexCat

	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}

		switch kv[0] {
		case "LEXID":
			lexCat.LexID = kv[1]
		case "STEM":
			lexCat.Stem = kv[1]
		case "ROOTCLASS":
			lexCat.RootClass = kv[1]
		case "INFLECTID":
			lexCat.InflictType = kv[1]
		}
	}

	entry.LexCat = lexCat
}

func parseVerb(attrs []xml.Attr, entry *DictionaryEntry) {
	var verb Verb
	var cpValue, parseValue string

	for _, attr := range attrs {
		switch attr.Name.Local {
		case "verb":
			verb.VerbType = attr.Value
		case "cp":
			cpValue = attr.Value
		case "parse":
			parseValue = attr.Value
		}
	}

	// Parse cp value for class and pada
	if cpValue != "" {
		parseCp(cpValue, &verb)
	}

	// Parse parse value
	if parseValue != "" {
		verb.Parse = strings.Split(parseValue, "+")
	}

	entry.Verb = verb
}

func parseCp(cp string, verb *Verb) {
	// Format: "1,10P" or "0Ā,0P" etc
	parts := strings.Split(cp, ",")
	for _, part := range parts {
		if part == "" {
			continue
		}

		// Extract class number and pada
		var classNum strings.Builder
		var pada string

		for _, ch := range part {
			if ch >= '0' && ch <= '9' {
				classNum.WriteRune(ch)
			} else if ch == 'P' {
				pada = "P"
			} else if ch == 'Ā' || ch == 'A' {
				pada = "A"
			}
		}

		if classNum.Len() > 0 {
			if n, err := strconv.Atoi(classNum.String()); err == nil && n > 0 {
				verb.VerbClass = n
			}
		}

		if pada != "" {
			verb.Pada = pada
		}
	}
}

func parseOtherSpellings(value string, entry *DictionaryEntry) {
	// Format: "L1,X1;L2,X2"
	pairs := strings.Split(value, ";")
	for _, pair := range pairs {
		parts := strings.SplitN(pair, ",", 2)
		if len(parts) == 2 {
			spelling := parts[1]
			if spelling != entry.Word {
				entry.Variants = append(entry.Variants, spelling)
			}
		}
	}
}

func readElementText(decoder *xml.Decoder) (string, error) {
	var text strings.Builder
	depth := 1

	for depth > 0 {
		token, err := decoder.Token()
		if err != nil {
			return text.String(), err
		}

		switch elem := token.(type) {
		case xml.StartElement:
			depth++
		case xml.EndElement:
			depth--
		case xml.CharData:
			text.Write(elem)
		}
	}

	return text.String(), nil
}

func skipElement(decoder *xml.Decoder) error {
	_, err := readElementText(decoder)
	return err
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
					slog.Info("processing entries", "done", entryCount)
				}
			}
		}
	}

	fmt.Printf("Conversion complete. Processed %d entries.\n", entryCount)
	return nil
}
