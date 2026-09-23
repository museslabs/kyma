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

// glamourMargin is the horizontal margin glamour reserves around a document,
// on top of the word wrap width it is configured with.
const glamourMargin = 2

type RendererOption func(*Renderer) error

type Renderer struct {
	tr      *glamour.TermRenderer
	wrapped map[int]*glamour.TermRenderer
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
	p.Register(Prioritized[Parser](NewGridParser(), 1))
	p.Register(Prioritized[Parser](NewGridRowParser(), 2))
	p.Register(Prioritized[Parser](NewGridColumnParser(), 3))

	r := &Renderer{
		tr:      tr,
		wrapped: map[int]*glamour.TermRenderer{},
		parser:  p,
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

// termRenderer returns a glamour renderer whose output is exactly width cells
// wide. Glamour lays its document out inside a two cell margin that it adds on
// top of the wrap width, so the wrap has to be asked for two cells short.
// Renderers are cached because a slide is re-rendered on every animation frame.
func (r *Renderer) termRenderer(width int) (*glamour.TermRenderer, error) {
	if width <= glamourMargin {
		return r.tr, nil
	}

	if tr, ok := r.wrapped[width]; ok {
		return tr, nil
	}

	tr, err := glamour.NewTermRenderer(
		glamour.WithStylePath(r.options.theme),
		glamour.WithWordWrap(width-glamourMargin),
	)
	if err != nil {
		return nil, err
	}
	r.wrapped[width] = tr

	return tr, nil
}

func (r *Renderer) Render(in string, animating bool, width, height int) (string, error) {
	return r.RenderBytes([]byte(in), animating, width, height)
}

func (r *Renderer) RenderBytes(in []byte, animating bool, width, height int) (string, error) {
	var b strings.Builder

	// Clear kitty images
	if !animating {
		b.WriteString("\x1b_Ga=d\x1b\\")
	}

	if err := r.renderNode(r.parser.Parse(in), animating, width, height, &b); err != nil {
		return "", err
	}

	return b.String(), nil
}

func (r *Renderer) renderNode(n Node, animating bool, width, height int, b *strings.Builder) error {
	if n == nil {
		return nil
	}

	switch n.Kind() {
	case NodeKindMarkdownRoot:
		break

	case NodeKindGlamour:
		n := n.(*GlamourNode)

		tr, err := r.termRenderer(width)
		if err != nil {
			return err
		}

		out, err := tr.Render(n.Text)
		if err != nil {
			return err
		}
		b.WriteString(out)

	case NodeKindError:
		n := n.(*ErrorNode)
		fmt.Fprintf(b, "%s\n", lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")).
			Render("⚠ "+n.Message))

	case NodeKindImage:
		n := n.(*ImageNode)

		// Keep an image from spilling out of the column it was placed in.
		imgWidth, imgHeight := n.Width, n.Height
		if width > 0 && imgWidth > width {
			if imgWidth > 0 {
				imgHeight = imgHeight * width / imgWidth
			}
			imgWidth = width
		}

		limg, err := r.options.imgBackend.Render(n.Path, imgWidth, imgHeight, true)
		if err != nil {
			fmt.Fprintf(b, "[Error rendering image: %s]", n.Label)
			break
		}

		if r.options.imgBackend.SymbolsOnly() {
			b.WriteString(limg)
			break
		}

		himg, err := r.options.imgBackend.Render(n.Path, imgWidth, imgHeight, false)
		if err != nil {
			fmt.Fprintf(b, "[Error rendering image: %s]", n.Label)
			break
		}

		if !animating {
			b.WriteString(ansi.SaveCursor)
			b.WriteString(himg)
			b.WriteString(ansi.RestoreCursor)
			b.WriteString(limg)
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

		codeWidth := 78
		if width > 0 {
			codeWidth = width
		}
		b.WriteString(lipgloss.NewStyle().Width(codeWidth).Render(renderedContent))
		b.WriteString("\n")

	case NodeKindGrid, NodeKindGridRow, NodeKindGridColumn:
		out, err := r.renderNodeBlock(n, animating, sizing{
			Width:  gridWidth(width),
			Budget: remainingHeight(height, b),
		})
		if err != nil {
			return err
		}
		b.WriteString(out)
		b.WriteString("\n")
		return nil

	default:
		return fmt.Errorf("invalid node kind: %d", n.Kind())
	}

	for _, c := range n.Children() {
		if err := r.renderNode(c, animating, width, height, b); err != nil {
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
