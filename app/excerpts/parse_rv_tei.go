package excerpts

import (
	"bufio"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/mahesh-hegde/dhee/app/common"
)

// XML structure definitions
type TEI struct {
	XMLName xml.Name `xml:"TEI"`
	Text    Text     `xml:"text"`
}

type Text struct {
	Body Body `xml:"body"`
}

type Body struct {
	Div Div `xml:"div"`
}

type P struct {
	Lang     string `xml:"lang,attr"`
	CharData string `xml:",chardata"`
}

type Div struct {
	XMLName xml.Name `xml:"div"`
	ID      string   `xml:"id,attr"`
	Type    string   `xml:"type,attr"`
	Divs    []Div    `xml:"div"`
	LGs     []LG     `xml:"lg"`
	Ps      []P      `xml:"p"`
}

type Stanza struct {
	ID         string      `xml:"id,attr"`
	Type       string      `xml:"type,attr"`
	Dedication *Dedication `xml:"dedication"`
	LGs        []LG        `xml:"lg"`
}

type Dedication struct {
	Addressee AddresseeGroup `xml:"addressee"`
	Group     Group          `xml:"group"`
}

type AddresseeGroup struct {
	PEng string `xml:"p"`
	Lang string `xml:"lang,attr"`
}

type Group struct {
	N    string `xml:"n,attr"`
	PEng string `xml:"p"`
}

type LG struct {
	ID     string `xml:"id,attr"`
	Lang   string `xml:"lang,attr"`
	Source string `xml:"source,attr"`
	Lines  []Line `xml:"l"`
}

type Line struct {
	ID      string `xml:"id,attr"`
	Lang    string `xml:"lang,attr"`
	Content string `xml:",chardata"`
	FS      []FS   `xml:"fs"`
	Words   []Word `xml:"w"`
}

type Word struct {
	ID      string `xml:"id,attr"`
	Content string `xml:",chardata"`
}

type FS struct {
	Type string `xml:"type,attr"`
	ID   string `xml:"id,attr"`
	F    []F    `xml:"f"`
}

type F struct {
	Name   string `xml:"name,attr"`
	String String `xml:"string"`
	Symbol Symbol `xml:"symbol"`
	FS     *FS    `xml:"fs"`
}

type String struct {
	Correction string `xml:"correction,attr"`
	Content    string `xml:",chardata"`
}

type Symbol struct {
	Value string `xml:"value,attr"`
}

type embeddingRelated struct {
	ReadableIndex string  `json:"readable_index"`
	Score         float32 `json:"score"`
}

type embeddingsForExcerpt struct {
	ReadableIndex string             `json:"readable_index"`
	Related       []embeddingRelated `json:"related"`
}

func ConvertRvTeiToExcerpts(file io.Reader) ([]Excerpt, error) {
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	var tei TEI
	if err := xml.Unmarshal(data, &tei); err != nil {
		return nil, fmt.Errorf("unmarshaling XML: %w", err)
	}

	var excerpts []Excerpt

	// Navigate to book level
	bookDiv := tei.Text.Body.Div
	var bookNum int
	fmt.Sscanf(bookDiv.ID, "b%d", &bookNum)

	// Iterate through hymns
	for _, hymnDiv := range bookDiv.Divs {
		hymnNum := extractNumber(hymnDiv.ID)

		// Extract dedication info if available
		var authors []string
		var meter string
		var group string
		var addressees []string

		// Iterate through stanzas
		for _, stanzaDiv := range hymnDiv.Divs {
			if stanzaDiv.Type == "dedication" {
				for _, subDiv := range stanzaDiv.Divs {
					if subDiv.Type == "addressee" {
						for _, p := range subDiv.Ps {
							if p.Lang == "eng" {
								addressees = append(addressees, p.CharData)
							}
						}
					}
					if subDiv.Type == "group" {
						for _, p := range subDiv.Ps {
							if p.Lang == "eng" {
								group = p.CharData
							}
						}
					}
				}
			}

			if stanzaDiv.Type != "stanza" {
				continue
			}

			stanzaNum := extractNumber(stanzaDiv.ID)
			path := []int{bookNum, hymnNum, stanzaNum}

			excerpt := Excerpt{
				ReadableIndex: fmt.Sprintf("%d.%d.%d", bookNum, hymnNum, stanzaNum),
				Path:          path,
				Authors:       authors,
				Meter:         meter,
				Addressees:    addressees,
				Group:         group,
				Auxiliaries:   make(map[string]Auxiliary),
			}

			// Parse all lg elements
			var zurLG, eichlerLG, vnhLG, lubLG, griffLG, oldLG, padaLG *LG

			for i := range stanzaDiv.LGs {
				lg := &stanzaDiv.LGs[i]
				switch lg.Source {
				case "zurich":
					zurLG = lg
				case "eichler":
					eichlerLG = lg
				case "vnh":
					vnhLG = lg
				case "lubotsky":
					lubLG = lg
				case "griffith":
					griffLG = lg
				case "oldenberg":
					oldLG = lg
				case "padapatha":
					padaLG = lg
				}
			}

			// Extract source texts
			if eichlerLG != nil {
				excerpt.SourceText = extractTextLines(eichlerLG)
			}

			if zurLG != nil {
				excerpt.RomanText = extractTextLines(zurLG)
			} else if lubLG != nil {
				excerpt.RomanText = extractTextLines(lubLG)
			} else if vnhLG != nil {
				excerpt.RomanText = extractTextLines(vnhLG)
			}

			// Extract auxiliaries
			if griffLG != nil {
				excerpt.Auxiliaries["griffith"] = Auxiliary{
					Text: extractTextLines(griffLG),
				}
			}

			if oldLG != nil {
				excerpt.Auxiliaries["oldenberg"] = Auxiliary{
					Text: extractTextLines(oldLG),
				}
			}

			if padaLG != nil {
				excerpt.Auxiliaries["pada"] = Auxiliary{
					Text: extractPadaText(padaLG),
				}
			}

			// Extract glossings from zurich
			if zurLG != nil {
				excerpt.Glossings = extractGlossings(zurLG)
			}

			excerpts = append(excerpts, excerpt)
		}
	}

	return excerpts, nil
}

func extractNumber(id string) int {
	parts := strings.Split(id, "_")
	if len(parts) < 2 {
		return 0
	}

	// Extract number from parts like "b02", "h001", "01"
	numStr := strings.TrimPrefix(parts[len(parts)-1], "h")
	numStr = strings.TrimPrefix(numStr, "b")

	num, _ := strconv.Atoi(numStr)
	return num
}

func extractTextLines(lg *LG) []string {
	var lines []string
	for _, line := range lg.Lines {
		content := strings.TrimSpace(line.Content)
		if content != "" {
			lines = append(lines, content)
		}
	}
	return lines
}

func extractPadaText(lg *LG) []string {
	var words []string
	for _, line := range lg.Lines {
		for _, word := range line.Words {
			content := strings.TrimSpace(word.Content)
			if content != "" {
				words = append(words, content)
			}
		}
	}
	// Return as single line with words
	if len(words) > 0 {
		return []string{strings.Join(words, " | ")}
	}
	return nil
}

func extractGlossings(lg *LG) [][]WordGlossing {
	var result [][]WordGlossing

	for _, line := range lg.Lines {
		if !strings.HasSuffix(line.ID, "_tokens") {
			continue
		}

		var lineGlossings []WordGlossing

		for _, fs := range line.FS {
			if fs.Type != "zurich_info" {
				continue
			}

			glossing := WordGlossing{}
			var modifiers []Modifier

			for _, f := range fs.F {
				switch f.Name {
				case "surface":
					glossing.Surface = f.String.Content
				case "gra_lemma":
					glossing.Lemma = f.String.Content
				case "gra_gramm":
					glossing.Gramm = f.Symbol.Value
				case "morphosyntax":
					if f.FS != nil {
						for _, morphF := range f.FS.F {
							switch morphF.Name {
							case "case":
								glossing.Case = morphF.Symbol.Value
							case "number":
								glossing.Number = morphF.Symbol.Value
							case "gender":
								glossing.Gender = morphF.Symbol.Value
							case "tense":
								glossing.Tense = morphF.Symbol.Value
							case "voice":
								glossing.Voice = morphF.Symbol.Value
							case "person":
								glossing.Person = morphF.Symbol.Value
							case "mood":
								glossing.Mood = morphF.Symbol.Value
							default:
								modifiers = append(modifiers, Modifier(morphF.Name+":"+morphF.Symbol.Value))
							}
						}
					}
				}
			}

			if len(modifiers) > 0 {
				glossing.Modifiers = modifiers
			}

			lineGlossings = append(lineGlossings, glossing)
		}

		if len(lineGlossings) > 0 {
			result = append(result, lineGlossings)
		}
	}

	return result
}

const (
	// Per-word weight for a lemma that is different from surface.
	textualLemmaWeight = 0.5
	// How many suggestions to generate per excerpt
	maxTextualSuggestions = 5
	// Minimum similarity score to be considered for suggestion
	minTextualSimilarity = 0.2
)

type textualSuggestion struct {
	idx   int
	score float32
}

func computeTextualSuggestions(excerpts []Excerpt) {
	if len(excerpts) == 0 {
		return
	}
	slog.Info("Computing textual suggestions using TF-IDF...")

	docTermFreqs := make([]map[string]float32, len(excerpts))
	lemmaDocFreq := make(map[string]int)
	surfaceToLemma := make(map[string]string)

	for i, excerpt := range excerpts {
		docTermFreqs[i] = make(map[string]float32)
		lemmasInDoc := make(map[string]bool)
		for _, lineGlossing := range excerpt.Glossings {
			for _, glossing := range lineGlossing {
				surface := common.NormalizeSurface(glossing.Surface)
				lemma := common.NormalizeLemma(glossing.Lemma)
				if lemma == "" {
					continue
				}

				docTermFreqs[i][surface] += 1.0
				if surface != lemma {
					docTermFreqs[i][lemma] += textualLemmaWeight
				}
				surfaceToLemma[surface] = lemma
				lemmasInDoc[lemma] = true
			}
		}
		for lemma := range lemmasInDoc {
			lemmaDocFreq[lemma]++
		}
	}

	numDocs := float64(len(excerpts))
	lemmaIDF := make(map[string]float32)
	for lemma, df := range lemmaDocFreq {
		lemmaIDF[lemma] = float32(math.Log(numDocs / float64(df)))
	}

	docTFIDF := make([]map[string]float32, len(excerpts))
	docMagnitudes := make([]float32, len(excerpts))
	for i, termFreqs := range docTermFreqs {
		docTFIDF[i] = make(map[string]float32)
		var magnitudeSq float32
		for term, tf := range termFreqs {
			var idf float32
			if lemma, ok := surfaceToLemma[term]; ok {
				idf = lemmaIDF[lemma]
			} else {
				idf = lemmaIDF[term]
			}
			tfidf := tf * idf
			docTFIDF[i][term] = tfidf
			magnitudeSq += tfidf * tfidf
		}
		docMagnitudes[i] = float32(math.Sqrt(float64(magnitudeSq)))
	}

	allSuggestions := make([][]textualSuggestion, len(excerpts))
	for i := 0; i < len(excerpts); i++ {
		for j := i + 1; j < len(excerpts); j++ {
			v1, v2 := docTFIDF[i], docTFIDF[j]
			mag1, mag2 := docMagnitudes[i], docMagnitudes[j]

			if mag1 == 0 || mag2 == 0 {
				continue
			}

			var dotProduct float32
			if len(v1) > len(v2) {
				v1, v2 = v2, v1
			}
			for term, tfidf1 := range v1 {
				if tfidf2, ok := v2[term]; ok {
					dotProduct += tfidf1 * tfidf2
				}
			}

			score := dotProduct / (mag1 * mag2)
			if score >= minTextualSimilarity {
				allSuggestions[i] = append(allSuggestions[i], textualSuggestion{idx: j, score: score})
				allSuggestions[j] = append(allSuggestions[j], textualSuggestion{idx: i, score: score})
			}
		}
	}

	for i := range excerpts {
		suggestions := allSuggestions[i]
		sort.Slice(suggestions, func(k, l int) bool {
			return suggestions[k].score > suggestions[l].score
		})

		for k := 0; k < len(suggestions) && k < maxTextualSuggestions; k++ {
			suggestion := suggestions[k]
			score := suggestion.score
			excerpts[i].SuggestedTextual = append(excerpts[i].SuggestedTextual, Related{
				Scripture:             "rigveda", // This is hardcoded for now.
				ReadableIndex:         excerpts[suggestion.idx].ReadableIndex,
				TextualRelevanceScore: &score,
				AutoGenerated:         true,
			})
		}
	}
	slog.Info("Finished computing textual suggestions.")
}

func WriteExcerptsToJsonL(excerpts []Excerpt, outPath string) error {
	file, err := os.OpenFile(outPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("opening output file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	for _, excerpt := range excerpts {
		if err := encoder.Encode(excerpt); err != nil {
			return fmt.Errorf("encoding excerpt: %w", err)
		}
	}

	return nil
}

func PreprocessRvDataset(teiInputDir, outputDir, embeddingsFile string) error {
	// Load embeddings
	embeddings := make(map[string][]embeddingRelated)
	if embeddingsFile != "" {
		file, err := os.Open(embeddingsFile)
		if err != nil {
			if os.IsNotExist(err) {
				slog.Warn("embeddings file not found, skipping", "path", embeddingsFile)
			} else {
				return fmt.Errorf("opening embeddings file: %w", err)
			}
		} else {
			defer file.Close()
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				var emb embeddingsForExcerpt
				if err := json.Unmarshal(scanner.Bytes(), &emb); err != nil {
					slog.Error("unmarshaling embeddings line, skipping", "err", err, "line", scanner.Text())
					continue
				}
				embeddings[emb.ReadableIndex] = emb.Related
			}
			if err := scanner.Err(); err != nil {
				return fmt.Errorf("reading embeddings file: %w", err)
			}
			slog.Info("Loaded embeddings for excerpts", "count", len(embeddings))
		}
	}
	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	outputPath := outputDir + "/rv.jsonl"

	// Truncate output file if it exists
	if err := os.Truncate(outputPath, 0); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("truncating output file: %w", err)
	}

	// Read directory entries
	entries, err := os.ReadDir(teiInputDir)
	if err != nil {
		return fmt.Errorf("reading input directory: %w", err)
	}

	// Process each TEI file matching the pattern
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasPrefix(name, "rv_book_") || !strings.HasSuffix(name, ".tei") {
			continue
		}

		filePath := teiInputDir + "/" + name
		slog.Info("Processing TEI", "input_file", filePath)

		// Open and process the file
		file, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("opening %s: %w", name, err)
		}

		excerpts, err := ConvertRvTeiToExcerpts(file)
		file.Close()

		if err != nil {
			return fmt.Errorf("converting %s: %w", name, err)
		}

		// Augment excerpts with related info
		if len(embeddings) > 0 {
			for i := range excerpts {
				excerpt := &excerpts[i]
				if related, ok := embeddings[excerpt.ReadableIndex]; ok {
					for _, r := range related {
						score := r.Score // a copy
						excerpt.SuggestedSemantic = append(excerpt.SuggestedSemantic, Related{
							Scripture:        "rigveda", // This is hardcoded for now.
							ReadableIndex:    r.ReadableIndex,
							CosineSimilarity: &score,
							AutoGenerated:    true,
						})
					}
				}
			}
		}

		computeTextualSuggestions(excerpts)

		// Append excerpts to output file
		if err := WriteExcerptsToJsonL(excerpts, outputPath); err != nil {
			return fmt.Errorf("writing excerpts from %s: %w", name, err)
		}

		slog.Info("Processed input TEI file", "n_excerpts", len(excerpts), "input_file", name)
	}

	return nil
}
