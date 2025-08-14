package markdown

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/museslabs/kyma/internal/config"
	"github.com/museslabs/kyma/internal/img"
)

type RendererOption func(*Renderer) error

type Renderer struct {
	tr      *glamour.TermRenderer
	parser  *MarkdownParser
	options rendererOptions
}

type rendererOptions struct {
	imgBackend img.ImageBackend
	theme      string
}

func NewRenderer(theme string, options ...RendererOption) (*Renderer, error) {
	tr, err := glamour.NewTermRenderer(glamour.WithStylePath(theme))
	if err != nil {
		return nil, err
	}

	p := NewMarkdownParser()
	p.Register(Prioritized[Parser](NewImageParser(), 1))
	p.Register(Prioritized[Parser](NewCodeBlockParser(), 1))

	r := &Renderer{
		tr:     tr,
		parser: p,
		options: rendererOptions{
			imgBackend: img.Get("chafa"),
			theme:      theme,
		},
	}
	for _, o := range options {
		if err := o(r); err != nil {
			return nil, err
		}
	}
	return r, nil
}

func (r *Renderer) Render(in string, animating bool) (string, error) {
	return r.RenderBytes([]byte(in), animating)
}

func (r *Renderer) RenderBytes(in []byte, animating bool) (string, error) {
	var b strings.Builder

	// Clear kitty images
	if !animating {
		b.WriteString("\x1b_Ga=d\x1b\\")
	}

	if err := r.renderNode(r.parser.Parse(in), animating, &b); err != nil {
		return "", err
	}

	return b.String(), nil
}

func (r *Renderer) renderNode(n Node, animating bool, b *strings.Builder) error {
	if n == nil {
		return nil
	}

	switch n.Kind() {
	case NodeKindMarkdownRoot:
		break

	case NodeKindGlamour:
		n := n.(*GlamourNode)
		out, err := r.tr.Render(n.Text)
		if err != nil {
			return err
		}
		b.WriteString(out)

	case NodeKindImage:
		n := n.(*ImageNode)

		limg, err := r.options.imgBackend.Render(n.Path, n.Width, n.Height, true)
		if err != nil {
			fmt.Fprintf(b, "[Error rendering image: %s]", n.Label)
			break
		}

		if r.options.imgBackend.SymbolsOnly() {
			b.WriteString(limg)
			break
		}

		himg, err := r.options.imgBackend.Render(n.Path, n.Width, n.Height, false)
		if err != nil {
			fmt.Fprintf(b, "[Error rendering image: %s]", n.Label)
			break
		}

		if !animating {
			b.WriteString(ansi.SaveCursor)
			b.WriteString(limg)
			b.WriteString(ansi.RestoreCursor)
			b.WriteString(himg)
		} else {
			b.WriteString(limg)
		}

	case NodeKindCodeBlock:
		n := n.(*CodeBlockNode)

		lines := strings.Split(n.Code, "\n")

		var renderedContent string
		if n.Language != "" {
			lexer := lexers.Get(n.Language)
			if lexer == nil {
				lexer = lexers.Fallback
			}
			lexer = chroma.Coalesce(lexer)
			style := config.GetChromaStyle(r.options.theme)

			renderedContent = r.renderHighlightedCode(n.Code, lines, n, lexer, style)
		} else {
			renderedContent = r.renderPlainCode(lines, n)
		}

		// Apply consistent styling
		codeStyle := lipgloss.NewStyle().Width(78)

		b.WriteString(codeStyle.Render(renderedContent))

	default:
		return fmt.Errorf("invalid node kind: %d", n.Kind())
	}

	for _, c := range n.Children() {
		if err := r.renderNode(c, animating, b); err != nil {
			return err
		}
	}

	return nil
}

func WithImageBackend(backend string) RendererOption {
	return func(r *Renderer) error {
		r.options.imgBackend = img.Get(backend)
		return nil
	}
}

func (r *Renderer) formatLineNumber(lineNum, width int) string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		PaddingRight(1)

	lineNumStr := strconv.Itoa(lineNum)
	paddedLineNum := fmt.Sprintf("%*s", width-1, lineNumStr)
	return style.Render(paddedLineNum)
}

func (r *Renderer) getLineNumberWidth(startLine, totalLines int) int {
	if totalLines == 0 {
		return 0
	}
	maxLineNum := startLine + totalLines - 1
	return len(strconv.Itoa(maxLineNum)) + 2
}

func (r *Renderer) renderPlainCode(lines []string, info *CodeBlockNode) string {
	var result strings.Builder
	lineNumberWidth := 0

	if info.ShowLineNumbers {
		lineNumberWidth = r.getLineNumberWidth(info.StartLine, len(lines))
	}

	for i, line := range lines {
		displayLineNum := info.StartLine + i

		if info.ShowLineNumbers {
			result.WriteString(r.formatLineNumber(displayLineNum, lineNumberWidth))
		}

		result.WriteString(line)

		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}

func (r *Renderer) renderHighlightedCode(
	content string,
	lines []string,
	info *CodeBlockNode,
	lexer chroma.Lexer,
	style *chroma.Style,
) string {
	formatter := formatters.Get("terminal256")
	if formatter == nil {
		return r.renderPlainCode(lines, info)
	}

	iterator, err := lexer.Tokenise(nil, content)
	if err != nil {
		return r.renderPlainCode(lines, info)
	}

	var formattedBuf strings.Builder
	if err := formatter.Format(&formattedBuf, style, iterator); err != nil {
		return r.renderPlainCode(lines, info)
	}

	formattedLines := strings.Split(formattedBuf.String(), "\n")

	// Ensure we have the same number of lines
	for len(formattedLines) < len(lines) {
		formattedLines = append(formattedLines, "")
	}
	if len(formattedLines) > len(lines) {
		formattedLines = formattedLines[:len(lines)]
	}

	var result strings.Builder
	lineNumberWidth := 0

	if info.ShowLineNumbers {
		lineNumberWidth = r.getLineNumberWidth(info.StartLine, len(lines))
	}

	for i, line := range lines {
		displayLineNum := info.StartLine + i
		relativeLineNum := i + 1

		if info.ShowLineNumbers {
			result.WriteString(r.formatLineNumber(displayLineNum, lineNumberWidth))
		}

		if r.shouldHighlightLine(relativeLineNum, info.Ranges) {
			formattedLine := ""
			if i < len(formattedLines) {
				formattedLine = strings.TrimRight(formattedLines[i], " \t\n\r")
			}
			if formattedLine == "" {
				formattedLine = line
			}
			result.WriteString(formattedLine)
		} else {
			result.WriteString(line)
		}

		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}

func (r *Renderer) shouldHighlightLine(lineNum int, ranges []CodeBlockLineRange) bool {
	if len(ranges) == 0 {
		return true // highlight all lines if no ranges specified
	}

	for _, r := range ranges {
		if lineNum >= r.Start && lineNum <= r.End {
			return true
		}
	}
	return false
}
