package docstore

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/mahesh-hegde/dhee/app/common"
	"github.com/mahesh-hegde/dhee/app/config"
	"github.com/mahesh-hegde/dhee/app/dictionary"
	"github.com/mahesh-hegde/dhee/app/transliteration"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

var scriptureContextKey = parser.NewContextKey()

// MarkdownConverter holds state for converting custom markdown.
type MarkdownConverter struct {
	dictStore      dictionary.DictStore
	transliterator *transliteration.Transliterator
	goldmark       goldmark.Markdown
	conf           *config.DheeConfig
}

// NewMarkdownConverter creates a new markdown converter.
func NewMarkdownConverter(d dictionary.DictStore, t *transliteration.Transliterator, conf *config.DheeConfig) *MarkdownConverter {
	mc := &MarkdownConverter{
		dictStore:      d,
		transliterator: t,
		conf:           conf,
	}

	ext := &dheeMarkdownExtension{mc: mc}
	mc.goldmark = goldmark.New(
		goldmark.WithExtensions(ext),
	)
	return mc
}

// ConvertToHTML converts custom markdown to HTML.
func (mc *MarkdownConverter) ConvertToHTML(text string, scripture config.ScriptureDefn) (string, error) {
	var buf bytes.Buffer
	ctx := parser.NewContext()
	ctx.Set(scriptureContextKey, scripture)
	if err := mc.goldmark.Convert([]byte(text), &buf, parser.WithContext(ctx)); err != nil {
		return "", err
	}
	return buf.String(), nil
}

type dheeMarkdownExtension struct {
	mc *MarkdownConverter
}

// Extend adds custom parsing to goldmark.
func (e *dheeMarkdownExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		parser.WithASTTransformers(
			util.Prioritized(&dheeASTTransformer{mc: e.mc}, 100),
		),
	)
}

type dheeASTTransformer struct {
	mc *MarkdownConverter
}

var linkRegex = regexp.MustCompile(`@([a-zA-Z0-9_-]+)#([0-9.]+)`)

// Transform traverses the Markdown AST and applies custom transformations.
func (t *dheeASTTransformer) Transform(node *ast.Document, reader text.Reader, pc parser.Context) {
	err := ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		textValue := string(n.Text(reader.Source()))
		slog.Debug("visiting node", "kind", n.Kind().String(), "text", textValue)

		switch n.Kind() {
		case ast.KindEmphasis:
			if n.(*ast.Emphasis).Level == 1 {
				var next ast.Node
				for c := n.FirstChild(); c != nil; c = next {
					next = c.NextSibling()
					if c.Kind() == ast.KindText {
						txt := c.(*ast.Text)
						wordHK := string(txt.Segment.Value(reader.Source()))
						if wordIAST, err := t.mc.transliterator.Convert(wordHK, common.Transliteration("hk"), common.TlIAST); err == nil {
							newNode := ast.NewString([]byte(wordIAST))
							n.ReplaceChild(n, c, newNode)
						}
					}
				}
			}
		case ast.KindCodeSpan:
			var content strings.Builder
			for c := n.FirstChild(); c != nil; c = c.NextSibling() {
				if textNode, ok := c.(*ast.Text); ok {
					content.Write(textNode.Segment.Value(reader.Source()))
				}
			}
			wordHK := content.String()
			if wordHK == "" {
				return ast.WalkContinue, nil
			}

			wordSLP1, err := t.mc.transliterator.Convert(wordHK, common.Transliteration("hk"), common.TlSLP1)
			if err != nil {
				// Fallback to just transliterating to IAST
				if wordIAST, err := t.mc.transliterator.Convert(wordHK, common.Transliteration("hk"), common.TlIAST); err == nil {
					newNode := ast.NewString([]byte(wordIAST))
					n.Parent().ReplaceChild(n.Parent(), n, newNode)
				}
				return ast.WalkContinue, nil
			}

			dictName := t.mc.conf.DefaultDict
			entries, _ := t.mc.dictStore.Get(context.Background(), dictName, []string{wordSLP1})

			wordIAST, _ := t.mc.transliterator.Convert(wordHK, common.Transliteration("hk"), common.TlIAST)
			if len(entries) > 0 {
				link := ast.NewLink()
				link.Destination = fmt.Appendf(nil, "/dictionaries/%s/words/%s", dictName, wordSLP1)
				link.SetAttributeString("style", []byte("text-decoration: underline;"))
				link.AppendChild(link, ast.NewString([]byte(wordIAST)))
				n.Parent().ReplaceChild(n.Parent(), n, link)
			} else {
				n.Parent().ReplaceChild(n.Parent(), n, ast.NewString([]byte(wordIAST)))
			}

		case ast.KindLink:
			link := n.(*ast.Link)
			destination := string(link.Destination)
			if strings.HasPrefix(destination, "@") {
				matches := linkRegex.FindStringSubmatch(destination)
				if len(matches) == 3 {
					scriptureName := matches[1]
					path := matches[2]
					newDestination := fmt.Sprintf("/scriptures/%s/excerpts/%s", scriptureName, path)
					link.Destination = []byte(newDestination)
				}
			}
		}
		return ast.WalkContinue, nil
	})
	if err != nil {
		slog.Error("error transforming markdown AST", "error", err)
	}
}
