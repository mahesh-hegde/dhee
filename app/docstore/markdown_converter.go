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
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
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
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(&dheeHTMLRenderer{mc: e.mc}, 500),
	))
}

var linkRegex = regexp.MustCompile(`@([a-zA-Z0-9_-]+)#([0-9.]+)`)

type dheeHTMLRenderer struct {
	mc *MarkdownConverter
}

// RegisterFuncs registers the custom rendering functions.
func (r *dheeHTMLRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindEmphasis, r.renderEmphasis)
	reg.Register(ast.KindCodeSpan, r.renderCodeSpan)
	reg.Register(ast.KindLink, r.renderLink)
}

func (r *dheeHTMLRenderer) renderEmphasis(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.Emphasis)
	if n.Level != 1 {
		if entering {
			if n.Level == 2 {
				_, _ = w.WriteString("<strong>")
			} else {
				_, _ = w.WriteString("<em>")
			}
		} else {
			if n.Level == 2 {
				_, _ = w.WriteString("</strong>")
			} else {
				_, _ = w.WriteString("</em>")
			}
		}
		return ast.WalkContinue, nil
	}

	if entering {
		wordHK := string(node.Text(source))
		if wordIAST, err := r.mc.transliterator.Convert(wordHK, common.Transliteration("hk"), common.TlIAST); err == nil {
			_, _ = w.WriteString("<em>")
			_, _ = w.Write(util.EscapeHTML([]byte(wordIAST)))
			_, _ = w.WriteString("</em>")
			return ast.WalkSkipChildren, nil
		}

		// Fallback for failed transliteration
		_, _ = w.WriteString("<em>")
	} else {
		_, _ = w.WriteString("</em>")
	}
	return ast.WalkContinue, nil
}

func (r *dheeHTMLRenderer) renderCodeSpan(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}

	wordHK := string(n.Text(source))
	if wordHK == "" {
		return ast.WalkSkipChildren, nil
	}

	wordIAST, err := r.mc.transliterator.Convert(wordHK, common.Transliteration("hk"), common.TlIAST)
	if err != nil {
		wordIAST = wordHK // fallback
	}

	wordSLP1, err := r.mc.transliterator.Convert(wordHK, common.Transliteration("hk"), common.TlSLP1)
	if err != nil {
		// Just render as text if we can't get SLP1 for dictionary lookup.
		_, _ = w.Write(util.EscapeHTML([]byte(wordIAST)))
		return ast.WalkSkipChildren, nil
	}

	dictName := r.mc.conf.DefaultDict
	entries, _ := r.mc.dictStore.Get(context.Background(), dictName, []string{wordSLP1})

	if len(entries) > 0 {
		url := fmt.Sprintf("/dictionaries/%s/words/%s", dictName, wordSLP1)
		_, _ = w.WriteString(`<a href="`)
		_, _ = w.Write(util.EscapeHTML(util.URLEscape([]byte(url), true)))
		_, _ = w.WriteString(`" style="text-decoration: underline;">`)
		_, _ = w.Write(util.EscapeHTML([]byte(wordIAST)))
		_, _ = w.WriteString("</a>")
	} else {
		_, _ = w.Write(util.EscapeHTML([]byte(wordIAST)))
	}

	return ast.WalkSkipChildren, nil
}

func (r *dheeHTMLRenderer) renderLink(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.Link)
	if entering {
		destination := string(n.Destination)
		if strings.HasPrefix(destination, "@") {
			matches := linkRegex.FindStringSubmatch(destination)
			if len(matches) == 3 {
				scriptureName := matches[1]
				path := matches[2]
				newDestination := fmt.Sprintf("/scriptures/%s/excerpts/%s", scriptureName, path)
				n.Destination = []byte(newDestination)
			}
		}

		_, _ = w.WriteString("<a href=\"")
		_, _ = w.Write(util.EscapeHTML(util.URLEscape(n.Destination, true)))
		_ = w.WriteByte('"')
		if n.Title != nil {
			_, _ = w.WriteString(" title=\"")
			_, _ = w.Write(util.EscapeHTML(n.Title))
			_ = w.WriteByte('"')
		}
		if n.Attributes() != nil {
			html.RenderAttributes(w, n)
		}
		_ = w.WriteByte('>')
	} else {
		_, _ = w.WriteString("</a>")
	}
	return ast.WalkContinue, nil
}
