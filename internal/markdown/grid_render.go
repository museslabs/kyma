package markdown

import (
	"log/slog"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/museslabs/kyma/internal/config"
)

// defaultGridWidth is used when a grid is rendered without a terminal width,
// which happens in tests and when a slide is measured before the first resize.
const defaultGridWidth = 80

// sizing is the space a grid container has been given.
type sizing struct {
	// Width is the exact number of cells the block must occupy.
	Width int
	// Budget is the vertical space the container may hand out to children that
	// asked for a share of it. It is zero when the height is unconstrained.
	Budget int
	// Height, when positive, is a height the block must occupy exactly. It is
	// only set by a parent that has already allocated the space.
	Height int
}

// renderGrid stacks a [GridNode]'s rows into a block that is exactly s.Width
// cells wide. The grid is only as tall as its rows unless a row claimed a share
// of the height budget, which is what lets a master layout fill the slide.
func (r *Renderer) renderGrid(n *GridNode, animating bool, s sizing) (string, error) {
	style := boxStyle(n.Box)
	inner := s.Width - style.GetHorizontalFrameSize()

	budget := 0
	if s.Budget > 0 {
		budget = s.Budget - style.GetVerticalFrameSize()
	}

	rows := n.Children()
	if len(rows) == 0 {
		return style.Render(fitBlock("", inner, 0, AlignDefault, AlignDefault, true)), nil
	}

	blocks := make([]string, len(rows))
	heights := make([]int, len(rows))
	sizes := make([]Size, len(rows))

	shared := false
	for i, row := range rows {
		var err error
		blocks[i], err = r.renderNodeBlock(row, animating, sizing{Width: inner, Budget: budget})
		if err != nil {
			return "", err
		}
		heights[i] = lipgloss.Height(blocks[i])

		if row, ok := row.(*GridRowNode); ok {
			sizes[i] = row.Size
			shared = shared || row.Size.Kind == SizeFraction
		}
	}

	// Only re-render when a row asked for a share of the slide; a stack of
	// naturally sized rows is measured correctly on the first pass.
	if shared && budget > 0 {
		allocated := distribute(budget, n.Gap, sizes, heights)
		for i, row := range rows {
			if allocated[i] == heights[i] || allocated[i] <= 0 {
				continue
			}
			var err error
			blocks[i], err = r.renderNodeBlock(row, animating, sizing{
				Width:  inner,
				Budget: allocated[i],
				Height: allocated[i],
			})
			if err != nil {
				return "", err
			}
		}
		heights = allocated
	}

	height := sum(heights) + n.Gap*(len(rows)-1)
	if forced := resolveSize(n.Size, budget); forced > 0 {
		height = forced
	}
	if s.Height > 0 {
		height = s.Height - style.GetVerticalFrameSize()
	}

	return style.Render(fitBlock(
		strings.Join(blocks, strings.Repeat("\n", n.Gap+1)),
		inner, height, n.Align, n.VAlign, true,
	)), nil
}

// renderGridRow lays a [GridRowNode]'s columns out side by side in a block that
// is exactly s.Width cells wide. Every column is padded to the height of the
// tallest one so their borders and backgrounds line up.
func (r *Renderer) renderGridRow(n *GridRowNode, animating bool, s sizing) (string, error) {
	style := boxStyle(n.Box)
	inner := s.Width - style.GetHorizontalFrameSize()

	budget := 0
	if s.Budget > 0 {
		budget = s.Budget - style.GetVerticalFrameSize()
	}

	cols := n.Children()
	if len(cols) == 0 {
		return style.Render(fitBlock("", inner, 0, AlignDefault, AlignDefault, true)), nil
	}

	// The row's own height, when something has already fixed it.
	height := 0
	if s.Height > 0 {
		height = s.Height - style.GetVerticalFrameSize()
	} else if forced := resolveSize(n.Size, budget); forced > 0 {
		height = forced
	}

	colBudget := budget
	if height > 0 {
		colBudget = height
	}

	sizes := make([]Size, len(cols))
	boxes := make([]Box, len(cols))
	frames := make([]lipgloss.Style, len(cols))

	for i, col := range cols {
		if col, ok := col.(*GridColumnNode); ok {
			boxes[i] = col.Box
		}
		frames[i] = boxStyle(boxes[i])

		// A column has no natural width, so one that did not ask for a size
		// takes an equal share of the row.
		sizes[i] = Size{Kind: SizeFraction, Value: 1}
		if boxes[i].Size.Kind != SizeAuto {
			sizes[i] = boxes[i].Size
		}
	}
	widths := distribute(inner, n.Gap, sizes, nil)

	// Render every column's content first so the row can take the height of its
	// tallest column, then pad the rest to match.
	contents := make([]string, len(cols))
	natural := 0

	for i, col := range cols {
		var err error
		contents[i], err = r.renderColumnContent(col, animating, sizing{
			Width:  widths[i] - frames[i].GetHorizontalFrameSize(),
			Budget: vertical(colBudget, frames[i]),
		})
		if err != nil {
			return "", err
		}

		natural = max(natural, lipgloss.Height(contents[i])+frames[i].GetVerticalFrameSize())
	}

	if height == 0 {
		height = natural
	}

	blocks := make([]string, 0, len(cols)*2)
	for i, col := range cols {
		if i > 0 && n.Gap > 0 {
			blocks = append(blocks, strings.Repeat(" ", n.Gap))
		}
		blocks = append(blocks, frames[i].Render(fitBlock(
			contents[i],
			widths[i]-frames[i].GetHorizontalFrameSize(),
			height-frames[i].GetVerticalFrameSize(),
			boxes[i].Align, boxes[i].VAlign,
			!containsKind(col, NodeKindImage),
		)))
	}

	return style.Render(fitBlock(
		lipgloss.JoinHorizontal(lipgloss.Top, blocks...),
		inner, height, n.Align, n.VAlign, true,
	)), nil
}

// renderColumnContent renders a column's children into a block that is exactly
// s.Width cells wide, without the column's own padding or border.
func (r *Renderer) renderColumnContent(n Node, animating bool, s sizing) (string, error) {
	if s.Width <= 0 {
		return "", nil
	}

	var b strings.Builder
	for _, c := range n.Children() {
		if err := r.renderNode(c, animating, s.Width, s.Budget, &b); err != nil {
			return "", err
		}
	}

	// Glamour opens every document with a blank line, and how many it adds
	// depends on the block that follows. Trimming them keeps neighbouring
	// columns starting on the same row whatever kind of content they hold.
	content := b.String()
	graphics := containsKind(n, NodeKindImage)
	if !graphics {
		content = trimBlankLines(content)
	}

	return fitBlock(content, s.Width, 0, AlignDefault, AlignDefault, !graphics), nil
}

// trimBlankLines drops the leading and trailing lines of a block that render as
// nothing. They cannot be trimmed as whitespace because a styled blank line is
// mostly escape sequences.
func trimBlankLines(s string) string {
	lines := strings.Split(s, "\n")

	start := 0
	for start < len(lines) && visibleWidth(lines[start]) == 0 {
		start++
	}

	end := len(lines)
	for end > start && visibleWidth(lines[end-1]) == 0 {
		end--
	}

	return strings.Join(lines[start:end], "\n")
}

// renderNodeBlock renders a grid container as a standalone block.
func (r *Renderer) renderNodeBlock(n Node, animating bool, s sizing) (string, error) {
	switch n := n.(type) {
	case *GridNode:
		return r.renderGrid(n, animating, s)

	case *GridRowNode:
		return r.renderGridRow(n, animating, s)

	case *GridColumnNode:
		style := boxStyle(n.Box)

		content, err := r.renderColumnContent(n, animating, sizing{
			Width:  s.Width - style.GetHorizontalFrameSize(),
			Budget: vertical(s.Budget, style),
		})
		if err != nil {
			return "", err
		}

		height := 0
		if s.Height > 0 {
			height = s.Height - style.GetVerticalFrameSize()
		}

		return style.Render(fitBlock(
			content,
			s.Width-style.GetHorizontalFrameSize(), height,
			n.Align, n.VAlign,
			!containsKind(n, NodeKindImage),
		)), nil

	default:
		var b strings.Builder
		if err := r.renderNode(n, animating, s.Width, s.Budget, &b); err != nil {
			return "", err
		}
		return fitBlock(b.String(), s.Width, s.Height, AlignDefault, AlignDefault, true), nil
	}
}

// fitBlock reshapes a rendered block so it is exactly width cells wide and,
// when height is positive, exactly that many lines tall. Short lines are padded
// according to halign and long ones are cut when truncate is set, which is what
// keeps columns from staircasing once they are joined side by side. Blocks that
// contain terminal graphics are never cut, since that would corrupt their
// escape sequences.
func fitBlock(s string, width, height int, halign, valign Align, truncate bool) string {
	if width <= 0 {
		return ""
	}

	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")

	if height > 0 {
		lines = fitLines(lines, height, valign)
	}

	indent := blockIndent(lines, width, halign)

	for i, line := range lines {
		if indent > 0 && visibleWidth(line) > 0 {
			line = strings.Repeat(" ", indent) + line
		}

		w := ansi.StringWidth(line)
		if w > width {
			if !truncate {
				continue
			}
			line, w = ansi.Truncate(line, width, ""), width
		}

		lines[i] = line + strings.Repeat(" ", width-w)
	}

	return strings.Join(lines, "\n")
}

// blockIndent is how far a block has to move right to sit at halign. The block
// moves as a whole rather than line by line, because glamour has already padded
// each of its lines out to the full width and centring those individually would
// do nothing at all. Only the ink counts: the margin glamour puts either side of
// a document is measured out so that centred text really is centred.
func blockIndent(lines []string, width int, halign Align) int {
	if halign != AlignCenter && halign != AlignEnd {
		return 0
	}

	left, right := width, 0
	for _, line := range lines {
		plain := strings.TrimRight(ansi.Strip(line), " \t")
		if plain == "" {
			continue
		}
		left = min(left, ansi.StringWidth(plain)-ansi.StringWidth(strings.TrimLeft(plain, " \t")))
		right = max(right, ansi.StringWidth(plain))
	}

	if right == 0 {
		return 0
	}

	pad := width - (right - left)
	if pad <= 0 {
		return 0
	}
	if halign == AlignCenter {
		pad /= 2
	}

	return max(pad-left, 0)
}

// visibleWidth is the width of a line up to its last non-blank cell, ignoring
// the trailing padding a renderer added to square the block off.
func visibleWidth(line string) int {
	return ansi.StringWidth(strings.TrimRight(ansi.Strip(line), " \t"))
}

// fitLines pads or trims a block to exactly height lines, placing the content
// it already has according to valign.
func fitLines(lines []string, height int, valign Align) []string {
	if len(lines) > height {
		switch valign {
		case AlignCenter:
			start := (len(lines) - height) / 2
			return lines[start : start+height]
		case AlignEnd:
			return lines[len(lines)-height:]
		default:
			return lines[:height]
		}
	}

	above := 0
	switch pad := height - len(lines); valign {
	case AlignCenter:
		above = pad / 2
	case AlignEnd:
		above = pad
	}

	out := make([]string, 0, height)
	out = append(out, make([]string, above)...)
	out = append(out, lines...)
	for len(out) < height {
		out = append(out, "")
	}

	return out
}

// resolveSize is the exact height a container claims out of budget. A fraction
// is a share handed out by the parent's [distribute] rather than something a
// container can resolve on its own, so it reports zero.
func resolveSize(s Size, budget int) int {
	switch s.Kind {
	case SizeCells:
		return int(s.Value)
	case SizePercent:
		if budget <= 0 {
			return 0
		}
		return int(s.Value / 100 * float64(budget))
	default:
		return 0
	}
}

// vertical is the height budget left inside style's border and padding.
func vertical(budget int, style lipgloss.Style) int {
	if budget <= 0 {
		return 0
	}
	return max(budget-style.GetVerticalFrameSize(), 0)
}

// boxStyle turns a [Box] into the lipgloss style that frames a container.
func boxStyle(box Box) lipgloss.Style {
	style := lipgloss.NewStyle()

	if box.PadX > 0 || box.PadY > 0 {
		style = style.Padding(box.PadY, box.PadX)
	}

	if border, ok := borderFor(box.Border); ok {
		color := box.BorderColor
		if color == "" {
			color = config.DefaultBorderColor
		}
		style = style.Border(border).BorderForeground(lipgloss.Color(color))
	}

	return style
}

func borderFor(name string) (lipgloss.Border, bool) {
	if name == "" || strings.EqualFold(name, "none") {
		return lipgloss.Border{}, false
	}

	border, ok := config.GetBorder(name)
	if !ok {
		slog.Warn("invalid layout border", slog.String("value", name))
	}

	return border, ok
}

// gridWidth falls back to a usable width when the renderer has not been told
// how wide the terminal is.
func gridWidth(width int) int {
	if width <= 0 {
		return defaultGridWidth
	}
	return width
}

// remainingHeight is the height budget left for a grid once the content already
// written above it on the slide is accounted for.
func remainingHeight(height int, b *strings.Builder) int {
	if height <= 0 {
		return 0
	}
	return max(height-(lipgloss.Height(b.String())-1), 0)
}

// containsKind reports whether n or any of its descendants is of kind.
func containsKind(n Node, kind NodeKind) bool {
	if n.Kind() == kind {
		return true
	}
	for _, c := range n.Children() {
		if containsKind(c, kind) {
			return true
		}
	}
	return false
}
