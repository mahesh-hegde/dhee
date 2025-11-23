package docstore

import (
	"bytes"
	"context"
	"fmt"
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

var scriptureContextKey = parser.NewContextKey("scripture")

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
	_ = ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch n.Kind() {
		case ast.KindEmphasis:
			if n.(*ast.Emphasis).Level == 1 {
				var next ast.Node
				for c := n.FirstChild(); c != nil; c = next {
					next = c.NextSibling()
					if c.Kind() == ast.KindText {
						txt := c.(*ast.Text)
						wordHK := string(txt.Segment.Value(reader.Source()))
						if !strings.Contains(wordHK, " ") {
							if wordIAST, err := t.mc.transliterator.Convert(wordHK, common.Transliteration("hk"), common.TlIAST); err == nil {
								newNode := ast.NewString([]byte(wordIAST))
								n.ReplaceChild(n, c, newNode)
							}
						}
					}
				}
			}
		case ast.KindCodeSpan:
			if textNode := n.FirstChild(); textNode != nil && textNode.Kind() == ast.KindText {
				txt := textNode.(*ast.Text)
				wordHK := string(txt.Segment.Value(reader.Source()))
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
					link.Destination = []byte(fmt.Sprintf("/dictionary/%s/%s", dictName, wordSLP1))
					link.SetAttributeString("style", []byte("text-decoration: underline;"))
					link.AppendChild(link, ast.NewString([]byte(wordIAST)))
					n.Parent().ReplaceChild(n.Parent(), n, link)
				} else {
					n.Parent().ReplaceChild(n.Parent(), n, ast.NewString([]byte(wordIAST)))
				}
			}

		case ast.KindText:
			txtNode := n.(*ast.Text)
			content := string(txtNode.Segment.Value(reader.Source()))
			if !strings.Contains(content, "@") {
				return ast.WalkContinue, nil
			}

			matches := linkRegex.FindAllStringSubmatchIndex(content, -1)
			if len(matches) == 0 {
				return ast.WalkContinue, nil
			}

			var newNodes []ast.Node
			lastIndex := 0
			for _, match := range matches {
				start, end := match[0], match[1]
				if start > lastIndex {
					newNodes = append(newNodes, ast.NewString([]byte(content[lastIndex:start])))
				}
				scriptureName := content[match[2]:match[3]]
				path := content[match[4]:match[5]]
				link := ast.NewLink()
				link.Destination = []byte(fmt.Sprintf("/scriptures/%s/%s", scriptureName, path))
				link.AppendChild(link, ast.NewString([]byte(fmt.Sprintf("@%s#%s", scriptureName, path))))
				newNodes = append(newNodes, link)
				lastIndex = end
			}
			if lastIndex < len(content) {
				newNodes = append(newNodes, ast.NewString([]byte(content[lastIndex:])))
			}

			parent := n.Parent()
			for _, newNode := range newNodes {
				parent.InsertBefore(parent, n, newNode)
			}
			parent.RemoveChild(parent, n)
			return ast.WalkSkipChildren, nil
		}
		return ast.WalkContinue, nil
	})
	return
}
