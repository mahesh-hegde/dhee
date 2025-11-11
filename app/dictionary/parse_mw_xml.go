package dictionary

import (
	"bufio"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/mahesh-hegde/dhee/app/common"
	"github.com/mahesh-hegde/dhee/app/transliteration"
)

type MwXmlEntryBody struct {
	Content string `xml:",innerxml"`
}

func attrVal(el xml.StartElement, attr string) string {
	for _, a := range el.Attr {
		if a.Name.Local == attr {
			return a.Value
		}
	}
	return ""
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

func xmlToDictionaryEntry(xml MwXmlEntry, lastPageNum string) (DictionaryEntry, Meaning) {
	tl := standardTl

	entry := DictionaryEntry{
		Word: xml.Header.Key1,
	}
	meaning := Meaning{
		Word:           entry.Word,
		PrintedPageNum: xml.Tail.PC,
		HTag:           xml.Tag,
		SId:            xml.Tail.L,
	}

	iast, err := tl.Convert(entry.Word, common.TlSLP1, common.TlIAST)
	if err == nil {
		entry.IAST = iast
	} else {
		slog.Debug("unable to convert word to IAST", "word", entry.Word)
	}

	if xml.Header.Key2 != "" && xml.Header.Key2 != entry.Word {
		meaning.Variants = []string{xml.Header.Key2}
	}

	// Use last page number if current entry doesn't have one
	if meaning.PrintedPageNum == "" {
		meaning.PrintedPageNum = lastPageNum
	}

	// Parse homonym number
	if xml.Header.Hom != "" {
		if num, err := strconv.Atoi(xml.Header.Hom); err == nil {
			meaning.HomonymNumber = num
		}
	}

	//	 Parse body into segments
	parseBody(xml.Body.Content, &meaning)

	// convert otherspellins to IAST for ease of lookup from canonical scriptures
	for _, va := range meaning.Variants {
		iast, err := tl.Convert(va, common.TlSLP1, common.TlIAST)
		if err != nil {
			slog.Debug("cannot convert variant to IAST", "word", va)
		} else {
			meaning.VariantsIAST = append(meaning.VariantsIAST, iast)
		}
	}

	return entry, meaning
}

func parseBody(body string, meaning *Meaning) {
	meaning.Body.Markup = body

	// Wrap in root element to create valid XML
	wrapped := "<root>" + body + "</root>"

	decoder := xml.NewDecoder(strings.NewReader(wrapped))
	decoder.Strict = false // tolerate malformed XML
	decoder.Token()

	var plainText strings.Builder

	if err := walkXMLTree(decoder, &plainText, meaning, 0); err != nil {
		slog.Warn("error parsing body", "error", err, "id", meaning.SId)
	}

	meaning.Body.Plain = strings.TrimSpace(plainText.String())
}

func walkXMLTree(decoder *xml.Decoder, plainText *strings.Builder, meaning *Meaning, depth int) error {
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
			if err := handleStartElement(elem, decoder, plainText, meaning, depth); err != nil {
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

func handleStartElement(elem xml.StartElement, decoder *xml.Decoder, plainText *strings.Builder, meaning *Meaning, depth int) error {
	tagName := elem.Name.Local
	switch tagName {
	case "ab":
		return handleAb(elem, decoder, plainText)

	case "s1":
		return handleS1(elem, decoder, plainText, meaning)

	case "s":
		return handleS(elem, decoder, plainText, meaning)

	case "ls":
		return handleLs(decoder, plainText, meaning)

	case "hom":
		return handleHom(decoder, plainText, meaning)

	case "etym", "lang", "bot", "bio", "ns", "i", "lex":
		return handleStripTag(decoder, plainText)

	case "info":
		return handleInfo(elem, meaning)

	case "pb", "div", "pcol":
		return handlePcol(decoder, plainText)

	case "shortlong", "srs":
		// These appear within <s> tags, skip them
		return nil

	default:
		// Recurse into unknown tags
		return walkXMLTree(decoder, plainText, meaning, depth+1)
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

func handleS1(el xml.StartElement, decoder *xml.Decoder, plainText *strings.Builder, meaning *Meaning) error {
	if slp1Attr := attrVal(el, "slp1"); slp1Attr != "" {
		meaning.Referenced = append(meaning.Referenced, slp1Attr)
	}

	content, err := readElementText(decoder)
	if err != nil {
		return err
	}
	plainText.WriteString(content)
	return nil
}

func handleS(_ xml.StartElement, decoder *xml.Decoder, plainText *strings.Builder, meaning *Meaning) error {
	content, err := readElementText(decoder)
	if err != nil {
		return err
	}
	// Remove if equal to word or in otherSpellings
	if content != meaning.Word {
		found := slices.Contains(meaning.Variants, content)
		if !found {
			meaning.Referenced = append(meaning.Referenced, content)
		}
		iast, err := standardTl.Convert(content, common.TlSLP1, common.TlIAST)
		if err != nil {
			slog.Warn("could not convert SLP string to IAST", "content", content)
		} else {
			plainText.WriteString(iast)
		}
	}
	return nil
}

func handleLs(decoder *xml.Decoder, plainText *strings.Builder, meaning *Meaning) error {
	content, err := readElementText(decoder)
	if err != nil {
		return err
	}
	plainText.WriteRune('[')
	plainText.WriteString(content)
	plainText.WriteRune(']')
	meaning.LitRefs = append(meaning.LitRefs, content)
	return nil
}

func handleHom(decoder *xml.Decoder, plainText *strings.Builder, meaning *Meaning) error {
	content, err := readElementText(decoder)
	if err != nil {
		return err
	}

	// Remove if at beginning and matches HomonymNumber
	trimmed := strings.TrimSuffix(strings.TrimSpace(content), ".")
	if meaning.HomonymNumber > 0 && trimmed == strconv.Itoa(meaning.HomonymNumber) {
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

func handleInfo(elem xml.StartElement, meaning *Meaning) error {
	for _, attr := range elem.Attr {
		switch attr.Name.Local {
		case "lex":
			meaning.LexicalGender = attr.Value

		case "lexcat":
			parseLexCat(attr.Value, meaning)

		case "verb":
			parseVerb(elem.Attr, meaning)

		case "or", "and", "orsl", "orwr":
			parseOtherSpellings(attr.Value, meaning)
		}
	}

	return skipElement(xml.NewDecoder(strings.NewReader("<dummy/>")))
}

func parseLexCat(value string, meaning *Meaning) {
	if value == "loan" {
		meaning.LexCat = LexCat{IsLoan: true}
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

	meaning.LexCat = lexCat
}

func parseVerb(attrs []xml.Attr, meaning *Meaning) {
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

	meaning.Verb = verb
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

func parseOtherSpellings(value string, meaning *Meaning) {
	// Format: "L1,X1;L2,X2"
	pairs := strings.Split(value, ";")
	for _, pair := range pairs {
		parts := strings.SplitN(pair, ",", 2)
		if len(parts) == 2 {
			spelling := parts[1]
			if spelling != meaning.Word {
				meaning.Variants = append(meaning.Variants, spelling)
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

var standardTl = func() *transliteration.Transliterator {
	tl, err := transliteration.NewTransliterator(transliteration.TlOptions{})
	if err != nil {
		log.Panicf("failed to instantiate transliterator: %s", err)
	}
	return tl
}()

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

	var lastEntry *DictionaryEntry

	writeLastEntry := func() error {
		if lastEntry != nil {
			// Write as JSON line
			jsonBytes, err := json.Marshal(lastEntry)
			lastEntry = nil
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
		return nil
	}

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
				dictEntry, meaning := xmlToDictionaryEntry(entry, lastPageNum)

				// Update last page number
				if meaning.PrintedPageNum != "" {
					lastPageNum = meaning.PrintedPageNum
				}

				if lastEntry != nil && lastEntry.Word == dictEntry.Word {
					lastEntry.Meanings = append(lastEntry.Meanings, meaning)
				} else {
					if err := writeLastEntry(); err != nil {
						return err
					}
					lastEntry = &dictEntry
					lastEntry.Meanings = append(lastEntry.Meanings, meaning)
				}
			}
		}
	}
	if err := writeLastEntry(); err != nil {
		return err
	}

	fmt.Printf("Conversion complete. Processed %d entries.\n", entryCount)
	return nil
}
