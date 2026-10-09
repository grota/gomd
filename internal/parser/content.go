package parser

import (
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// Image represents an image reference in a markdown document.
type Image struct {
	Alt   string
	Src   string
	Title string
}

// extractPlainText extracts plain text from an inline node.
func extractPlainText(n ast.Node, source []byte) string {
	var sb strings.Builder
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		if t, ok := child.(*ast.Text); ok {
			sb.Write(t.Segment.Value(source))
		} else {
			sb.WriteString(extractPlainText(child, source))
		}
	}
	return sb.String()
}

// ExtractImages extracts all images from markdown content.
func ExtractImages(content string) []Image {
	source := []byte(content)
	doc := goldmark.New().Parser().Parse(text.NewReader(source))

	var images []Image
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if img, ok := n.(*ast.Image); ok {
			images = append(images, Image{
				Alt:   extractPlainText(img, source),
				Src:   string(img.Destination),
				Title: string(img.Title),
			})
		}
		return ast.WalkContinue, nil
	})
	return images
}
