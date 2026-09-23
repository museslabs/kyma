package markdown

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"strings"
)

// GridParser parses a [grid] container.
//
// A grid stacks rows vertically:
//
//	[grid gap=1]
//	[row]
//	[col span=2]
//	master
//	[/col]
//	[col]
//	stack
//	[/col]
//	[/row]
//	[/grid]
//
// Rows are optional. Columns written straight inside a [grid] share one
// implicit row, which is the shape most slides want:
//
//	[grid]
//	[col]left[/col]
//	[col]right[/col]
//	[/grid]
type GridParser struct{}

func NewGridParser() *GridParser {
	return &GridParser{}
}

func (p GridParser) Trigger() []byte {
	return []byte{'['}
}

func (p GridParser) Parse(r *bytes.Reader, m *MarkdownParser) Node {
	attrs, ok := openTag(r, "grid")
	if !ok {
		return nil
	}

	grid := &GridNode{Box: parseBox(attrs, "height")}
	parseChildren(r, m, grid, "grid")
	normalizeGrid(grid)

	return grid
}

// GridRowParser parses a [row] container, which lays its columns out side by
// side. Content written straight inside a row shares one implicit column, so
// [row]slave[/row] is a complete row.
type GridRowParser struct{}

func NewGridRowParser() *GridRowParser {
	return &GridRowParser{}
}

func (p GridRowParser) Trigger() []byte {
	return []byte{'['}
}

func (p GridRowParser) Parse(r *bytes.Reader, m *MarkdownParser) Node {
	attrs, ok := openTag(r, "row")
	if !ok {
		return nil
	}

	row := &GridRowNode{Box: parseBox(attrs, "height")}
	parseChildren(r, m, row, "row")
	normalizeRow(row)

	return row
}

// GridColumnParser parses a [col] container, spelled either [col] or [column].
// A column holds slide content and may nest a further [grid] to split its space.
type GridColumnParser struct{}

func NewGridColumnParser() *GridColumnParser {
	return &GridColumnParser{}
}

func (p GridColumnParser) Trigger() []byte {
	return []byte{'['}
}

func (p GridColumnParser) Parse(r *bytes.Reader, m *MarkdownParser) Node {
	attrs, ok := openTag(r, "col", "column")
	if !ok {
		return nil
	}

	col := &GridColumnNode{Box: parseBox(attrs, "width")}
	parseChildren(r, m, col, "col", "column")

	return col
}

// parseChildren fills parent with everything up to its closing tag. Text that
// no registered [Parser] claims is collected into [GlamourNode]s.
//
// A closing tag that belongs to an enclosing container ends the parent
// implicitly and is left in the reader for that container to consume, so a
// missing [/col] costs the author a column rather than the rest of the slide.
// Running out of input closes the parent too, and records an [ErrorNode] so the
// mistake is visible on the slide.
func parseChildren(r *bytes.Reader, m *MarkdownParser, parent Node, names ...string) {
	var chunk bytes.Buffer

	flush := func() {
		if text := strings.Trim(chunk.String(), " \t\n"); text != "" {
			appendChild(parent, &GlamourNode{Text: text})
		}
		chunk.Reset()
	}

	for {
		b, err := r.ReadByte()
		if err != nil {
			flush()
			if !errors.Is(err, io.EOF) {
				slog.Warn("failed to advance reader", slog.Any("error", err))
			}
			appendChild(parent, &ErrorNode{
				Message: fmt.Sprintf("unclosed [%s]", names[0]),
			})
			return
		}

		if b == '[' {
			mark, _ := r.Seek(0, io.SeekCurrent)

			if closing, isClose := closeTag(r); isClose {
				flush()
				if !slices.Contains(names, closing) {
					// Belongs to an ancestor: rewind onto the '[' and let it
					// close this container implicitly.
					_, _ = r.Seek(mark-1, io.SeekStart)
				}
				return
			}

			_, _ = r.Seek(mark, io.SeekStart)
		}

		if n := m.parseNode(r, b); n != nil {
			flush()
			appendChild(parent, n)
			continue
		}

		if fence, ok := fencedBlock(r, b); ok {
			chunk.WriteString(fence)
			continue
		}

		chunk.WriteByte(b)
	}
}

// fencedBlock copies a code fence through verbatim so that layout tags inside
// one are shown rather than acted on. It reports false for anything that is not
// the start of a fence.
func fencedBlock(r *bytes.Reader, b byte) (string, bool) {
	if b != '`' || !atLineStart(r) {
		return "", false
	}

	fence, ok := scanFence(r)
	if !ok {
		return "", false
	}

	return string(b) + fence, true
}

// normalizeGrid guarantees every child of a grid is a row. A run of columns
// written straight inside the grid shares one implicit row, while loose content
// gets a full width row of its own so it stays where the author put it. The
// grid's gap carries over to rows that did not set one, so [grid gap=2] spaces
// its columns out too.
func normalizeGrid(grid *GridNode) {
	var (
		out     []Node
		pending Node
		columns bool
	)

	for _, c := range grid.Children() {
		if c.Kind() == NodeKindGridRow || c.Kind() == NodeKindError {
			pending = nil
			c.SetParent(grid)
			out = append(out, c)
			continue
		}

		isColumn := c.Kind() == NodeKindGridColumn
		if pending == nil || columns != isColumn {
			columns = isColumn
			pending = &GridRowNode{}
			pending.SetParent(grid)
			out = append(out, pending)
		}
		appendChild(pending, c)
	}
	grid.children = out

	for _, row := range grid.children {
		row, ok := row.(*GridRowNode)
		if !ok {
			continue
		}
		if row.Gap == 0 {
			row.Gap = grid.Gap
		}
		normalizeRow(row)
	}
}

// normalizeRow guarantees every child of a row is a column, wrapping runs of
// loose content in a single implicit column.
func normalizeRow(row *GridRowNode) {
	row.children = wrapChildren(row, row.children, NodeKindGridColumn, func() Node {
		return &GridColumnNode{}
	})
}

// wrapChildren groups every run of children that is not already of kind into a
// container built by newContainer.
func wrapChildren(parent Node, children []Node, kind NodeKind, newContainer func() Node) []Node {
	if len(children) == 0 {
		return children
	}

	var (
		out     []Node
		pending Node
	)

	for _, c := range children {
		if c.Kind() == kind {
			pending = nil
			c.SetParent(parent)
			out = append(out, c)
			continue
		}

		// Errors are reported by the container that raised them rather than
		// pushed into a wrapper of their own.
		if c.Kind() == NodeKindError {
			pending = nil
			c.SetParent(parent)
			out = append(out, c)
			continue
		}

		if pending == nil {
			pending = newContainer()
			pending.SetParent(parent)
			out = append(out, pending)
		}
		appendChild(pending, c)
	}

	return out
}

func appendChild(parent, child Node) {
	child.SetParent(parent)
	parent.AddChild(child)
}
